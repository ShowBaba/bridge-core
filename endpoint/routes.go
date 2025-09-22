package endpoint

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

func InitializeEndpointRoutes(app fiber.Router, db *gorm.DB, qC *amqp091.Connection) {
	repo := NewRepository(db)
	applicationSvc := application.NewService(application.NewRepository(db), qC, audit.NewService(audit.NewRepository(db)))
	databaseSvc := database.NewService(database.NewRepository(db), applicationSvc, audit.NewService(audit.NewRepository(db)), qC)
	svc := NewService(repo, applicationSvc, databaseSvc, audit.NewService(audit.NewRepository(db)))
	h := NewHandler(svc)

	r := app.Group("/endpoint")

	r.Post("/execute/:identifier", h.execute)

	rAuth := r.Group("")
	rAuth.Use(utils.ValidateAuthHeaderToken())
	rAuth.Post("/create", h.create)
	rAuth.Patch("/:endpoint_id/update", h.update)

}
