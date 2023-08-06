package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

var ctx = context.Background()

func InitNotificationQueue(connection *amqp091.Connection) error {
	log.Println("setting up notification tasks queue")
	channel, err := connection.Channel()
	if err != nil {
		return err
	}

	defer channel.Close()

	queue, err := channel.QueueDeclare(
		utils.NOTIFICATION_QUEUE,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	emailMsgs, err := channel.Consume(
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

	forever := make(chan bool)
	go func() {
		for {
			select {
			case emailMsg := <-emailMsgs:
				var payload EmailMsgPayload
				err := json.Unmarshal(emailMsg.Body, &payload)
				if err != nil {
					log.Println(err)
					// TODO: store logs for application
				}
				fmt.Println("processing notification ... ")
				if err := HandleEmailMsg(ctx, payload); err != nil {
					log.Println(err)
				}
			}
		}
	}()
	<-forever
	return nil
}
