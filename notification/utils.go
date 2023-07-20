package notification

import (
	"net/smtp"

	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func SendEmail(mail utils.Mail) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	message := mail.BuildMessage()
	auth := smtp.PlainAuth("", utils.GetConfig().MailUsername, utils.GetConfig().MailPassword, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, mail.Sender, mail.To, []byte(message))
}
