package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/models"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

var (
	pgDbCLient  *gorm.DB
	mongoClient *mongo.Client
)

func InitDBQueue(pg *gorm.DB, mongo *mongo.Client, connection *amqp091.Connection) error {
	log.Println("setting up database tasks queue")

	pgDbCLient = pg
	mongoClient = mongo
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	err = channel.ExchangeDeclare(
		utils.DATABASE_QUEUE,
		amqp091.ExchangeTopic,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare an exchange: %v", err)
	}

	queue, err := channel.QueueDeclare(
		utils.DATABASE_QUEUE,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	err = channel.QueueBind(
		queue.Name,
		"",
		utils.DATABASE_QUEUE,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind the queue to the exchange: %v", err)
	}

	databaseTasks, err := channel.Consume(
		queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("error subscribing to message - %v", err)
	}

	var (
		TotalWorkers = 10
		requestCh    = make(chan utils.DatabaseTask, 20)
		errorCh      = make(chan error, TotalWorkers)
		waitGroup    = &sync.WaitGroup{}
	)

	// Start worker pool
	for i := 0; i < TotalWorkers; i++ {
		worker := &Worker{
			id:        i,
			waitGroup: waitGroup,
			requestCh: requestCh,
			errorCh:   errorCh,
		}
		go worker.run()
		go func(workerID int, errorCh chan error) {
			for err := range errorCh {
				log.Printf("Error from worker %v: %s", workerID, err)
				// TODO: store logs for application
			}
		}(i, errorCh)
	}

	for databaseTask := range databaseTasks {
		var task utils.DatabaseTask
		err := json.Unmarshal(databaseTask.Body, &task)
		if err != nil {
			log.Println("error unmarshalling message:", err)
			continue
		}
		requestCh <- task
	}

	go func(errorCh chan error) {
		for err := range errorCh {
			log.Printf("Error from worker: %s", err)
			// TODO: store logs for application
		}
	}(errorCh)

	go func() {
		waitGroup.Wait()
		close(requestCh)
		close(errorCh)
	}()

	return nil
}

type Worker struct {
	id        int
	waitGroup *sync.WaitGroup
	requestCh chan utils.DatabaseTask
	errorCh   chan error
}

func (w Worker) run() {
	w.waitGroup.Add(1)

	for task := range w.requestCh {
		log.Printf("Worker [%d] processing task with db id: (%v)", w.id, task.DatabaseID)
		switch task.Action {
		case utils.FetchDBAction:
			err := handleFetchDbTask(task, pgDbCLient, mongoClient)
			if err != nil {
				// TODO: send notification to user, add endpoint to re-fetch db data
				log.Printf("error processing [%s] task: %v, from worker: %v", utils.FetchDBAction, err, w.id)
				w.errorCh <- err
			}
		case utils.DeleteApplicationResourceAction:
			err := handleDeleteApplicationResourceTask(task, pgDbCLient, mongoClient)
			if err != nil {
				log.Printf("error processing [%s] task: %v, from worker: %v", utils.DeleteApplicationResourceAction, err, w.id)
				w.errorCh <- err
			}
		case utils.DeleteDBResourceAction:
			err := handleDeleteDBResourceTask(task, pgDbCLient, mongoClient)
			if err != nil {
				log.Printf("error processing [%s] task: %v, from worker: %v", utils.DeleteDBResourceAction, err, w.id)
				w.errorCh <- err
			}
		}
	}

	w.waitGroup.Done()
}

func handleFetchDbTask(task utils.DatabaseTask, pgDb *gorm.DB, mongoClient *mongo.Client) error {
	var database *models.Database
	database, _, err := database.FetchDatabase(pgDb, models.Database{ID: task.DatabaseID})
	if err != nil {
		return fmt.Errorf("failed to fetch database: %v", err)
	}

	rawPassword, err := utils.Decrypt(database.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %v", err)
	}

	appDbPg, err := utils.TestDatabaseConnection(utils.DatabaseConnectionPayload{
		Host:     database.Host,
		Port:     database.Port,
		Database: database.Database,
		Username: database.Username,
		Password: string(rawPassword),
		DbEngine: database.DbEngine,
	})
	if err != nil {
		return fmt.Errorf("failed to test database connection: %v", err)
	}

	sqlLogCh := make(chan string)
	errorCh := make(chan error)
	resultCh := make(chan utils.SchemaData)

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		FetchDatabaseInfo(appDbPg, sqlLogCh, resultCh, errorCh)
	}()

	go func() {
		defer wg.Done()
		LogSqlQuery(mongoClient, database.ApplicationID, task.UserID, sqlLogCh, errorCh)
	}()

	go func() {
		defer wg.Done()
		StoreData(pgDb, task.DatabaseID, task.UserID, resultCh, errorCh)
	}()

	go func() {
		wg.Wait()
		close(sqlLogCh)
		close(errorCh)
		close(resultCh)
	}()

	return nil
}

func handleDeleteApplicationResourceTask(task utils.DatabaseTask, pgDb *gorm.DB, monogoClient *mongo.Client) error {
	var endpoint models.Endpoint
	var endpointIDs []uint
	endpoints, err := endpoint.FetchEndpoints(db, models.Endpoint{ApplicationID: task.ApplicationID})
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		endpointIDs = append(endpointIDs, endpoint.ID)
	}
	if err := endpoint.DeleteMany(db, endpointIDs); err != nil {
		return err
	}

	var database models.Database
	databases, err := database.FetchDatabases(db, models.Database{ApplicationID: task.ApplicationID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var (
		databaseIDs []uint
		schemaIDs   []uint
		tableIDs    []uint
		columnIDs   []uint
	)

	for _, database := range databases {
		databaseIDs = append(databaseIDs, database.ID)

		var schema models.Schema
		schemas, err := schema.FetchSchemas(db, models.Schema{DatabaseID: database.ID})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for _, schema := range schemas {
			schemaIDs = append(schemaIDs, schema.ID)

			var table models.Table
			tables, err := table.FetchTables(db, models.Table{SchemaID: schema.ID})
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			for _, table := range tables {
				tableIDs = append(tableIDs, table.ID)

				var column models.Column
				columns, err := column.FetchColumns(db, models.Column{TableID: table.ID})
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}

				for _, column := range columns {
					columnIDs = append(columnIDs, column.ID)
				}
			}
		}
	}

	var column *models.Column
	if err := column.DeleteMany(db, columnIDs); err != nil {
		return err
	}

	var table *models.Table
	if err := table.DeleteMany(db, tableIDs); err != nil {
		return err
	}

	var schema *models.Schema
	if err := schema.DeleteMany(db, schemaIDs); err != nil {
		return err
	}

	if err := database.DeleteMany(db, databaseIDs); err != nil {
		return err
	}
	return nil
}

func handleDeleteDBResourceTask(task utils.DatabaseTask, pgDb *gorm.DB, monogoClient *mongo.Client) error {
	var (
		schemaIDs []uint
		tableIDs  []uint
		columnIDs []uint
	)

	var schema models.Schema
	schemas, err := schema.FetchSchemas(db, models.Schema{DatabaseID: task.DatabaseID})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	for _, schema := range schemas {
		schemaIDs = append(schemaIDs, schema.ID)

		var table models.Table
		tables, err := table.FetchTables(db, models.Table{SchemaID: schema.ID})
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for _, table := range tables {
			tableIDs = append(tableIDs, table.ID)

			var column models.Column
			columns, err := column.FetchColumns(db, models.Column{TableID: table.ID})
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			for _, column := range columns {
				columnIDs = append(columnIDs, column.ID)
			}
		}
	}

	var column *models.Column
	if err := column.DeleteMany(db, columnIDs); err != nil {
		return err
	}

	var table *models.Table
	if err := table.DeleteMany(db, tableIDs); err != nil {
		return err
	}

	if err := schema.DeleteMany(db, schemaIDs); err != nil {
		return err
	}

	return nil
}
