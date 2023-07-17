package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/utils"
)

var ctx = context.Background()

func InitNotificationQueue(connection *amqp091.Connection) {
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}

	defer channel.Close()

	err = channel.ExchangeDeclare(
		utils.NOTIFICATION_QUEUE,
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
		utils.NOTIFICATION_QUEUE,
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
		utils.NOTIFICATION_QUEUE,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind the queue to the exchange: %v", err)
	}

	emailMsgs, err := channel.Consume(
		utils.NOTIFICATION_QUEUE,
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
			case emailMsg := <-emailMsgs:
				var payload EmailMsgPayload
				err := json.Unmarshal(emailMsg.Body, &payload)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("processing notification ... ")
				if err := HandleEmailMsg(ctx, payload); err != nil {
					log.Fatal(err)
				}
			}
		}
	}()
	<-forever
}
