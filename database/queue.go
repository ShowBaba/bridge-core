package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
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

	databaseTasks, err := channel.Consume(
		utils.DATABASE_QUEUE,
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

				wg.Add(1)
				go func() {
					defer wg.Done()
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
						Name:     database.Name,
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
					fmt.Println("procesing database task: ", task.DatabaseID)
					FetchSchemaTables(appDbPg, sqlLogCh, resultCh, errorCh)
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					LogSqlQuery(mongoClient, sqlLogCh, errorCh)
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					StoreData(pgDb, task.DatabaseID, resultCh, errorCh)
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

