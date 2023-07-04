package notification

import (
	"net/smtp"

	"github.com/showbaba/query-bridge/bridge/utils"
	"github.com/showbaba/query-bridge/shared"
)

func SendEmail(mail shared.Mail) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	message := mail.BuildMessage()
	auth := smtp.PlainAuth("", utils.GetConfig().MailUsername, utils.GetConfig().MailPassword, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, mail.Sender, mail.To, []byte(message))
}
