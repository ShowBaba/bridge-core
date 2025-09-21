package application

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"gorm.io/gorm"

	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func InitializeApplicationRoutes(app *fiber.App, db *gorm.DB, qC *amqp091.Connection) {
	repo := NewRepository(db)
	svc := NewService(repo, qC, audit.NewService(audit.NewRepository(db)))
	h := NewHandler(svc)

	r := app.Group("/application", utils.ValidateAuthHeaderToken())

	r.Post("/create", h.createApplication)
	r.Patch("/:application_id/update", h.updateApplication)
	r.Delete("/:application_id/delete", h.deleteApplication)
}
