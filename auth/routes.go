package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"gorm.io/gorm"
)

func InitializeAuthRoutes(app fiber.Router, db *gorm.DB, qC *amqp091.Connection) {
	userSvc := user.NewService(user.NewRepository(db), qC)
	svc := NewService(userSvc, audit.NewService(audit.NewRepository(db)))
	h := NewHandler(svc)
	r := app.Group("/auth")

	r.Post("/login", h.login)
}
