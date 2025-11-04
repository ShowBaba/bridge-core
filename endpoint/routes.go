package endpoint

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/log"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func InitializeEndpointRoutes(app fiber.Router, db *gorm.DB, qC *amqp091.Connection, mongo *mongo.Client) {
	repo := NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	applicationSvc := application.NewService(application.NewRepository(db), qC, auditSvc)
	databaseSvc := database.NewService(database.NewRepository(db), applicationSvc, auditSvc, qC)
	logSvc := log.NewService(log.NewRepository(mongo))
	svc := NewService(repo, applicationSvc, databaseSvc, auditSvc, logSvc)
	h := NewHandler(svc)

	r := app.Group("/endpoint")

	rAuth := r.Group("")
	rAuth.Use(utils.ValidateAuthHeaderToken())
	rAuth.Post("/create", h.create)
	rAuth.Patch("/:endpoint_id/update", h.update)
	rAuth.Delete("/:endpoint_id/delete", h.delete)
	rAuth.Post("/preview-sql", h.previewSQL)
	rAuth.Post("/scripts/preview", h.previewScript)
	rAuth.Patch("/:endpoint_id/scripts", h.updateScripts)

	app.All("/api/:version<v[0-9]+>/:app_slug/*", h.execute)
}
