package database

import (
	"encoding/json"
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
		return fmt.Errorf("failed to declare an exchange: %v", err)
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
		return err
	}

	err = channel.QueueBind(
		queue.Name,
		"",
		utils.DATABASE_QUEUE,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind the queue to the exchange: %v", err)
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
		return err
	}

	var (
		TOTAL_WORKERS = 10
		requestCh     = make(chan utils.DatabaseTask, TOTAL_WORKERS)
		errorCh       = make(chan error, 1) // handles error from a worker
		waitGroup     = &sync.WaitGroup{}
	)

	// Start worker pool
	for i := 0; i < TOTAL_WORKERS; i++ {
		worker := &Worker{
			id:        i,
			waitGroup: waitGroup,
			requestCh: requestCh,
			quitCh:    make(chan bool, 2),
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
	quitCh    chan bool
}

func (w Worker) run() {
	log.Printf("worker [%v] running", w.id)
	w.waitGroup.Add(1)
	defer w.waitGroup.Done()

Loop:
	for {
		select {
		case task, ok := <-w.requestCh:
			if !ok {
				break Loop
			} else {
				err := handleTask(task, pgDbCLient, mongoClient)
				if err != nil {
					// TODO: send notification to user, add endpoint to re-fetch db data
					log.Printf("error processing task: %v, from worker: %v", err, w.id)
					w.errorCh <- err
				}
			}
		case <-w.quitCh:
			break Loop
		}
	}
}

func handleTask(task utils.DatabaseTask, pgDb *gorm.DB, mongoClient *mongo.Client) error {
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
