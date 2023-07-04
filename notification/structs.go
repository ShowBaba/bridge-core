package notification

type EmailMsgPayload struct {
	To      []string
	Subject string
	Body    string
}
