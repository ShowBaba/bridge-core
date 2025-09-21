package database

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

func InitializeDatabaseRoutes(app fiber.Router, db *gorm.DB, qC *amqp091.Connection) {
	repo := NewRepository(db)
	applicationSvc := application.NewService(application.NewRepository(db), qC, audit.NewService(audit.NewRepository(db)))
	svc := NewService(repo, applicationSvc, audit.NewService(audit.NewRepository(db)), qC)
	h := NewHandler(svc)

	r := app.Group("/database", utils.ValidateAuthHeaderToken())

	r.Patch("/:database_id/update", h.update)
	r.Delete("/:database_id/delete", h.delete)
	r.Post("/:application_id/add-database", h.add)

}
