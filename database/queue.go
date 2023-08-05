package database

import (
	"context"
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

func InitDBQueue(pgDb *gorm.DB, mongoClient *mongo.Client, connection *amqp091.Connection) error {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numWorkers := 10
	taskCh := make(chan utils.DatabaseTask)
	wg := sync.WaitGroup{}
	taskWg := sync.WaitGroup{}

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processTask(&wg, &taskWg, ctx, taskCh, pgDb, mongoClient)
	}

	for databaseTask := range databaseTasks {
		var task utils.DatabaseTask
		err := json.Unmarshal(databaseTask.Body, &task)
		if err != nil {
			log.Println("error unmarshalling message:", err)
			continue
		}
		taskCh <- task
	}

	log.Printf("Clean up")
	cancel()  // signal workers to stop
	wg.Wait() // wait for all workers to finish

	return nil
}

func processTask(wg *sync.WaitGroup, taskWg *sync.WaitGroup, ctx context.Context, taskCh <-chan utils.DatabaseTask, pgDb *gorm.DB, mongoClient *mongo.Client) {
	defer wg.Done()
Loop:
	for {
		select {
		case task, ok := <-taskCh:
			if !ok {
				break Loop
			}

			taskWg.Add(1)
			go func(task utils.DatabaseTask) {
				defer taskWg.Done()
				err := handleTask(ctx, task, pgDb, mongoClient)
				if err != nil {
					log.Printf("Error processing task: %v", err)
					// TODO: send notification to user, add endpoint to re-fetch db data
				}
			}(task)
		case <-ctx.Done():
			break Loop
		}
	}
}

func handleTask(ctx context.Context, task utils.DatabaseTask, pgDb *gorm.DB, mongoClient *mongo.Client) error {
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
		FetchSchemaTables(appDbPg, sqlLogCh, resultCh, errorCh)
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
