package queues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	databasePkg "github.com/showbaba/query-bridge/bridge-core/database"
	logPkg "github.com/showbaba/query-bridge/bridge-core/database-log"
	endpointPkg "github.com/showbaba/query-bridge/bridge-core/endpoint"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Queue struct {
	pgDbClient  *gorm.DB
	mongoClient *mongo.Client
	qConn       *amqp091.Connection
	databaseSvc databasePkg.Service
	endpointSvc endpointPkg.Service
	logSvc      logPkg.Service
	ctx         context.Context
}

func NewQueue(pg *gorm.DB, mongo *mongo.Client, connection *amqp091.Connection,
	databaseSvc databasePkg.Service, endpointSvc endpointPkg.Service,
	logSvc logPkg.Service) *Queue {
	q := &Queue{
		pgDbClient:  pg,
		mongoClient: mongo,
		qConn:       connection,
		databaseSvc: databaseSvc,
		endpointSvc: endpointSvc,
		logSvc:      logSvc,
		ctx:         context.Background(),
	}
	if err := q.initDBQueue(); err != nil {
		panic(err)
	}
	return q
}

func (q *Queue) initDBQueue() error {
	log.Println("setting up database tasks queue")

	ch, err := q.qConn.Channel()
	if err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(utils.DATABASE_QUEUE, "topic", false, false, false, false, nil); err != nil {
		return err
	}

	queue, err := ch.QueueDeclare(utils.DATABASE_QUEUE, false, false, false, false, nil)
	if err != nil {
		return err
	}

	if err := ch.QueueBind(queue.Name, "", utils.DATABASE_QUEUE, false, nil); err != nil {
		return err
	}

	databaseTasks, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	const totalWorkers = 10
	requestCh := make(chan utils.DatabaseTask, 2*totalWorkers)
	errorCh := make(chan error, 2*totalWorkers)

	for i := 0; i < totalWorkers; i++ {
		worker := &Worker{
			id:          i,
			requestCh:   requestCh,
			errorCh:     errorCh,
			pgDbClient:  q.pgDbClient,
			mongoClient: q.mongoClient,
			databaseSvc: q.databaseSvc,
			queue:       q,
		}
		go worker.run()
	}

	go func() {
		for err := range errorCh {
			log.Printf("worker error: %s", err)
		}
	}()

	// producer
	go func() {
		for msg := range databaseTasks {
			var task utils.DatabaseTask
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				log.Println("error unmarshalling message:", err)
				continue
			}
			requestCh <- task
		}
		close(requestCh)
		close(errorCh)
	}()

	return nil
}

type Worker struct {
	id          int
	requestCh   chan utils.DatabaseTask
	errorCh     chan error
	pgDbClient  *gorm.DB
	mongoClient *mongo.Client
	databaseSvc databasePkg.Service
	queue       *Queue
}

func (w *Worker) run() {
	for task := range w.requestCh {
		log.Printf("Worker [%d] processing task with db id: (%v)", w.id, task.DatabaseID)
		var err error
		switch task.Action {
		case utils.FetchDBAction:
			err = w.queue.handleFetchDbTask(task)
		case utils.DeleteApplicationResourceAction:
			err = w.queue.handleDeleteApplicationResourceTask(task)
		case utils.DeleteDBResourceAction:
			err = w.queue.handleDeleteDBResourceTask(task)
		}
		if err != nil {
			select {
			case w.errorCh <- err:
			default:
				log.Printf("worker %d dropping error: %v", w.id, err)
			}
		}
	}
}

func (q *Queue) handleFetchDbTask(task utils.DatabaseTask) error {
	dbRec, err := q.databaseSvc.Get(q.ctx, &databasePkg.Database{ID: task.DatabaseID})
	if err != nil {
		return fmt.Errorf("failed to fetch database: %v", err)
	}

	rawPassword, err := utils.Decrypt(dbRec.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %v", err)
	}

	appDbPg, err := utils.TestDatabaseConnection(utils.DatabaseConnectionPayload{
		Host:     dbRec.Host,
		Port:     dbRec.Port,
		Database: dbRec.Database,
		Username: dbRec.Username,
		Password: string(rawPassword),
		DbEngine: dbRec.DbEngine,
	})
	if err != nil {
		return fmt.Errorf("failed to test database connection: %v", err)
	}

	sqlLogCh := make(chan string, 32)
	errCh := make(chan error, 32)
	resultCh := make(chan utils.SchemaData, 32)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		fetchDatabaseInfo(appDbPg, sqlLogCh, resultCh, errCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		q.logSqlQuery(dbRec.ApplicationID, task.UserID, sqlLogCh, errCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		storeData(q.pgDbClient, task.DatabaseID, task.UserID, resultCh, errCh)
	}()

	// drain errors so producers never block
	done := make(chan struct{})
	go func() {
		for e := range errCh {
			log.Printf("pipeline error: %v", e)
		}
		close(done)
	}()

	go func() {
		wg.Wait()
		close(sqlLogCh)
		close(resultCh)
		close(errCh)
	}()

	<-done
	return nil
}

func (q *Queue) logSqlQuery(applicationID, userID string, ch <-chan string, errCh chan<- error) {
	for logData := range ch {
		err := q.logSvc.Create(logData, applicationID, userID)
		if err != nil {
			errCh <- err
		}
	}
}

func (q *Queue) handleDeleteApplicationResourceTask(task utils.DatabaseTask) error {
	var endpointIDs []string
	endpoints, err := q.endpointSvc.List(q.ctx, endpointPkg.Endpoint{ApplicationID: task.ApplicationID}, utils.ListOpts{})
	if err != nil {
		return err
	}
	for _, e := range endpoints {
		endpointIDs = append(endpointIDs, e.ID)
	}
	if err := q.endpointSvc.DeleteMany(q.ctx, endpointIDs); err != nil {
		return err
	}

	databases, err := q.databaseSvc.List(q.ctx, databasePkg.Database{ApplicationID: task.ApplicationID}, utils.ListOpts{})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var (
		databaseIDs []string
		schemaIDs   []string
		tableIDs    []string
		columnIDs   []string
	)

	for _, d := range databases {
		databaseIDs = append(databaseIDs, d.ID)

		schemas, err := q.databaseSvc.ListSchemas(q.ctx, databasePkg.Schema{DatabaseID: d.ID})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for _, s := range schemas {
			schemaIDs = append(schemaIDs, s.ID)

			tables, err := q.databaseSvc.ListTables(q.ctx, databasePkg.Table{SchemaID: s.ID})
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			for _, t := range tables {
				tableIDs = append(tableIDs, t.ID)

				columns, err := q.databaseSvc.ListColumns(q.ctx, databasePkg.Column{TableID: t.ID})
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}

				for _, c := range columns {
					columnIDs = append(columnIDs, c.ID)
				}
			}
		}
	}

	if err := q.databaseSvc.DeleteManyColumns(q.ctx, columnIDs); err != nil {
		return err
	}
	if err := q.databaseSvc.DeleteManyTables(q.ctx, tableIDs); err != nil {
		return err
	}
	if err := q.databaseSvc.DeleteManySchemas(q.ctx, schemaIDs); err != nil {
		return err
	}
	if err := q.databaseSvc.DeleteMany(q.ctx, databaseIDs); err != nil {
		return err
	}
	return nil
}

func (q *Queue) handleDeleteDBResourceTask(task utils.DatabaseTask) error {
	var (
		schemaIDs []string
		tableIDs  []string
		columnIDs []string
	)

	schemas, err := q.databaseSvc.ListSchemas(q.ctx, databasePkg.Schema{DatabaseID: task.DatabaseID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	for _, s := range schemas {
		schemaIDs = append(schemaIDs, s.ID)

		tables, err := q.databaseSvc.ListTables(q.ctx, databasePkg.Table{SchemaID: s.ID})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for _, t := range tables {
			tableIDs = append(tableIDs, t.ID)

			columns, err := q.databaseSvc.ListColumns(q.ctx, databasePkg.Column{TableID: t.ID})
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			for _, c := range columns {
				columnIDs = append(columnIDs, c.ID)
			}
		}
	}

	if err := q.databaseSvc.DeleteManyColumns(q.ctx, columnIDs); err != nil {
		return err
	}
	if err := q.databaseSvc.DeleteManyTables(q.ctx, tableIDs); err != nil {
		return err
	}
	if err := q.databaseSvc.DeleteManySchemas(q.ctx, schemaIDs); err != nil {
		return err
	}

	return nil
}
