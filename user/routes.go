package user

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func InitializeUserRoutes(app fiber.Router, dbClient *gorm.DB, qC *amqp091.Connection) {
	repo := NewRepository(dbClient)
	svc := NewService(repo, qC)
	h := NewHandler(svc)

	r := app.Group("/user")

	r.Post("/register", h.Register)
	r.Get("/get-profile", utils.ValidateAuthHeaderToken(), h.GetProfile)
}
