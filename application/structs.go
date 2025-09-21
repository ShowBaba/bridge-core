package application

type CreateApplicationPayload struct {
	Name   string `json:"name" validate:"required"`
	ApiKey string `json:"api_key"`
}

type UpdateApplicationPayload struct {
	Name   string `json:"name"`
	ApiKey string `json:"api_key"`
}
