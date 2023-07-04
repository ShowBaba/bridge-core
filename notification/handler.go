package notification

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/shared"
)

func HandleEmailMsg(ctx context.Context, channel *amqp.Channel, payload EmailMsgPayload) error {
	mail := shared.Mail{
		Sender:  shared.MAIL_USERNAME,
		Subject: payload.Subject,
		To:      payload.To,
		Body:    payload.Body,
	}
	log.Println("sending email to - ", mail.To)
	if err := SendEmail(mail); err != nil {
		log.Printf("error while sending mail; err: %v", err)
	}

	return nil
}
