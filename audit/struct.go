package audit

type LogInput struct {
	UserID        string
	ApplicationID string
	Action        string
	EntityType    string
	EntityID      string
	Description   string
	Metadata      any
	IPAddress     string
}
