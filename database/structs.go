package database

type UpdateDatabasePayload struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbEngine string `json:"db_engine"`
}
