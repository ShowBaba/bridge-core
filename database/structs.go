package database

type UpdateDatabasePayload struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbEngine string `json:"db_engine"`
}
