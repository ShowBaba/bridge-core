package database

type UpdateDatabasePayload struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbEngine string `json:"db_engine"`
	Name     string `json:"name"`
}

type AddDatabasePayload struct {
	Name     string `json:"name" validate:"required"`
	Host     string `json:"host" validate:"required"`
	Port     uint   `json:"port" validate:"required"`
	Database string `json:"database" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	DbEngine string `json:"db_engine" validate:"required"`
}
