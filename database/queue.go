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

var ctx = context.Background()

func InitDBQueue(pgDb *gorm.DB, mongoClient *mongo.Client, connection *amqp091.Connection) {
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
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

	forever := make(chan bool)
	go func() {
		for {
			select {
			case databaseTask := <-databaseTasks:
				var (
					task     utils.DatabaseTask
					wg       sync.WaitGroup
					sqlLogCh = make(chan string)
					errorCh  = make(chan error)
					resultCh = make(chan utils.SchemaData)
				)
				err := json.Unmarshal(databaseTask.Body, &task)
				if err != nil {
					log.Fatal(err)
				}
				var database *models.Database
				database, _, err = database.FetchDatabase(pgDb, models.Database{ID: task.DatabaseID})
				if err != nil {
					errorCh <- err
					return
				}
				rawPassword, err := utils.Decrypt(database.Password, []byte(utils.GetConfig().EncryptionKey))
				if err != nil {
					errorCh <- err
					return
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
					errorCh <- err
					return
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					fmt.Println("procesing database task: ", task.DatabaseID)
					FetchSchemaTables(appDbPg, sqlLogCh, resultCh, errorCh)
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					LogSqlQuery(mongoClient, database.ApplicationID, task.UserID, sqlLogCh, errorCh)
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					StoreData(pgDb, task.DatabaseID, resultCh, errorCh)
				}()

				wg.Add(1)
				go func() {
					for err := range errorCh {
						if err != nil {
							log.Fatal(err)
						}
					}
				}()

				go func() {
					wg.Wait()
					close(sqlLogCh)
					close(errorCh)
					close(resultCh)
				}()

			}
		}

	}()

	<-forever
}
