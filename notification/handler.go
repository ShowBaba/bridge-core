package notification

import (
	"context"
	"log"

	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func HandleEmailMsg(ctx context.Context, payload EmailMsgPayload) error {
	mail := utils.Mail{
		Sender:  utils.MAIL_USERNAME,
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
