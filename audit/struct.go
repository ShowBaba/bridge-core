package audit

type LogInput struct {
	UserID        string
	ApplicationID string
	Action        string
	EntityType    string
	EntityID      string
	Description   string
	Severity      Severity
	Metadata      any
	IPAddress     string
}
