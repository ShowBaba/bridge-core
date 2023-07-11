package application

type CreateApplicationPayload struct {
	Name string `json:"name" validate:"required"`
}

type AddDatabasePayload struct {
	Name          string `json:"name" validate:"required"`
	Host          string `json:"host" validate:"required"`
	Port          uint   `json:"port" validate:"required"`
	Database      string `json:"database" validate:"required"`
	Username      string `json:"username" validate:"required"`
	Password      string `json:"password" validate:"required"`
	DbEngine      string `json:"db_engine" validate:"required"`
}
