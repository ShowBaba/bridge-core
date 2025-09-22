package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/showbaba/query-bridge/bridge-core/audit"
	logPkg "github.com/showbaba/query-bridge/bridge-core/database-log"
	"github.com/showbaba/query-bridge/bridge-core/queues"
	"golang.org/x/sync/errgroup"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/graphql-go/graphql"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/auth"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/db"
	"github.com/showbaba/query-bridge/bridge-core/endpoint"
	gql "github.com/showbaba/query-bridge/bridge-core/graphql"
	"github.com/showbaba/query-bridge/bridge-core/notification"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"github.com/showbaba/query-bridge/bridge-core/websocket"
	"go.mongodb.org/mongo-driver/mongo"

	"gorm.io/gorm"
)

func main() {
	var (
		g, ctx      = errgroup.WithContext(context.TODO())
		qConn       *amqp091.Connection
		dbClient    *gorm.DB
		pgConn      *sql.DB
		mongoClient *mongo.Client
		err         error
	)

	qConn, err = amqp091.Dial(utils.GetConfig().RabbitmqServerURL)
	if err != nil {
		log.Fatal(fmt.Errorf(`error opening queue connection; %v`, err))
	}

	dbClient, pgConn, err = db.ConnectToPgDB(
		utils.GetConfig().DbHost, utils.GetConfig().DbUser, utils.GetConfig().DbPassword, utils.GetConfig().DbName, utils.GetConfig().DbPort,
	)
	if err != nil {
		log.Fatal(fmt.Errorf(`error creating pg database connection; %v`, err))
	}

	mongoClient, err = db.ConnectToMongoDB(context.Background(), utils.GetConfig().MongoURI)
	if err != nil {
		log.Fatal(fmt.Errorf(`error creating mongo database connection; %v`, err))
	}

	g.Go(func() error {
		auditSvc := audit.NewService(audit.NewRepository(dbClient))
		applicationSvc := application.NewService(application.NewRepository(dbClient), qConn, auditSvc)
		databaseSvc := database.NewService(database.NewRepository(dbClient), applicationSvc, auditSvc, qConn)
		endpointSvc := endpoint.NewService(endpoint.NewRepository(dbClient), applicationSvc, databaseSvc, auditSvc)
		logSvc := logPkg.NewService(logPkg.NewRepository(mongoClient))
		if err := queues.NewQueue(dbClient, mongoClient, qConn, databaseSvc, endpointSvc, logSvc); err != nil {
			return fmt.Errorf(`error initializing database queue; %v`, err)
		}
		return nil
	})

	g.Go(func() error {
		fmt.Println("initializing notification queue...")
		if err := notification.InitNotificationQueue(qConn); err != nil {
			return fmt.Errorf(`err initilizing notification queue; %v`, err)
		}
		return nil
	})

	g.Go(func() error {
		fmt.Println("initializing websocket queue...")
		schema, err := graphql.NewSchema(
			graphql.SchemaConfig{
				Query: gql.Init(dbClient),
			},
		)
		if err != nil {
			return fmt.Errorf(`error creating schema; %v`, err)
		}

		app := fiber.New()
		app.Use(logger.New())
		app.Use(func(c *fiber.Ctx) error {
			ctx := utils.ContextWithIP(c.UserContext(), c.IP())
			c.SetUserContext(ctx)
			return c.Next()
		})
		corsSettings := cors.New(cors.Config{
			AllowOriginsFunc: func(origin string) bool {
				return true
			}, AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH",
			AllowHeaders:     "Origin, Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Requested-With",
			ExposeHeaders:    "Origin",
			AllowCredentials: true,
		})
		app.Use(corsSettings)

		// apitoolkitCfg := apitoolkit.Config{
		// 	RedactHeaders:     []string{"Content-Type", "Authorization", "Bearer"},
		// 	RedactRequestBody: []string{"$.password"},
		// 	APIKey:            utils.GetConfig().APIToolKitAPIKey,
		// }
		// apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkitCfg)
		// if err != nil {
		// 	return fmt.Errorf(`fail to initialize api tool kit; %v`, err)
		// }
		// app.Use(func(c *fiber.Ctx) error {
		// 	return apitoolkitClient.FiberMiddleware(c)
		// })

		InitializeRoutes(context.Background(), app, dbClient, qConn, mongoClient, schema)
		fmt.Println("starting server ....")
		port := utils.GetConfig().Port
		if port == "" {
			port = "8080"
		}
		log.Printf("starting server on port: %s", port)
		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			return fmt.Errorf(`fail to start server; %v`, err)
		}
		return nil
	})

	g.Go(func() error {
		return db.Migrate(dbClient)
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
	case <-ctx.Done():
		if err := qConn.Close(); err != nil {
			log.Printf("Error closing queue connection: %v", err)
		} else {
			log.Println("Queue connection closed.")
		}

		if err := pgConn.Close(); err != nil {
			log.Printf("Error closing pg connection: %v", err)
		} else {
			log.Println("PostgresSQL connection closed.")
		}

		if err := db.CloseDBConnection(mongoClient, ctx); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		} else {
			log.Println("MongoDB connection closed.")
		}
	default:
		log.Fatal("some unknown error occurred during shutdown")
	}

	log.Println("Server and database connections closed. Goodbye!")
}

func InitializeRoutes(ctx context.Context, app *fiber.App, dbCl *gorm.DB, qConnection *amqp091.Connection,
	mongoClient *mongo.Client, gqlSchema graphql.Schema) {

	app.Options("/gql", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	app.Post("/gql", func(c *fiber.Ctx) error {
		return gql.RunGQL(c, gqlSchema, ctx)
	})

	app.Use("/stream", websocket.StreamHandler(ctx, mongoClient))

	app.Get("/ping", func(c *fiber.Ctx) error {
		response := utils.APIResponse{
			Status:  http.StatusOK,
			Message: "bridge says pong!",
		}
		return c.JSON(response)
	})

	auth.InitializeAuthRoutes(app, dbCl, qConnection)
	user.InitializeUserRoutes(app, dbCl, qConnection)
	application.InitializeApplicationRoutes(app, dbCl, qConnection)
	database.InitializeDatabaseRoutes(app, dbCl, qConnection)
	endpoint.InitializeEndpointRoutes(app, dbCl, qConnection)
}
