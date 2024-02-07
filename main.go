package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/showbaba/query-bridge/bridge-core/queues"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

	mongoClient, err = db.ConnectToMongoDB(ctx, utils.GetConfig().MongoURI)
	if err != nil {
		log.Fatal(fmt.Errorf(`error creating mongo database connection; %v`, err))
	}

	g.Go(func() error {
		if err := queues.InitDBQueue(dbClient, mongoClient, qConn); err != nil {
			return fmt.Errorf(`error initializing database queue; %v`, err)
		}
		return nil
	})

	g.Go(func() error {
		if err := notification.InitNotificationQueue(qConn); err != nil {
			return fmt.Errorf(`err initilizing notification queue; %v`, err)
		}
		return nil
	})

	g.Go(func() error {
		schema, err := graphql.NewSchema(
			graphql.SchemaConfig{
				Query: gql.Init(dbClient),
			},
		)
		if err != nil {
			return fmt.Errorf(`error creating schema; %v`, err)
		}
		router := mux.NewRouter()
		InitializeRoutes(ctx, router, dbClient, qConn, mongoClient, schema)
		port := utils.GetConfig().Port
		if port == "" {
			port = "8080"
		}
		log.Printf("starting server on port: %s", port)
		if err := Run(router, fmt.Sprintf(`:%s`, port)); err != nil {
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

func InitializeRoutes(ctx context.Context, router *mux.Router, dbCl *gorm.DB, qConnection *amqp091.Connection,
	mongoClient *mongo.Client, gqlSchema graphql.Schema) {
	// graphql route
	router.HandleFunc("/gql", func(w http.ResponseWriter, r *http.Request) {
		gql.RunGQL(w, r, gqlSchema, ctx)
	}).Methods("POST", "OPTIONS")

	// log stream route
	router.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		websocket.StreamHandler(w, r, ctx, mongoClient)
	})

	router.HandleFunc("/ping", ping).Methods("GET")

	authRoutes := router.PathPrefix("/auth").Subrouter()
	auth.InitializeAuthRoutes(authRoutes, dbCl)

	userRoutes := router.PathPrefix("/user").Subrouter()
	user.InitializeUserRoutes(userRoutes, dbCl, qConnection)

	applicationRoutes := router.PathPrefix("/application").Subrouter()
	application.InitializeApplicationRoutes(applicationRoutes, dbCl, qConnection)

	databaseRoutes := router.PathPrefix("/database").Subrouter()
	database.InitializeApplicationRoutes(databaseRoutes, dbCl, qConnection)

	endpointRoutes := router.PathPrefix("/endpoint").Subrouter()
	endpoint.InitializeEndpointRoutes(endpointRoutes, dbCl, mongoClient)
}

func Run(r *mux.Router, host string) error {
	return http.ListenAndServe(
		host,
		handlers.CORS(
			handlers.AllowCredentials(),
			handlers.AllowedMethods([]string{"POST", "GET", "PUT", "OPTIONS", "DELETE", "PATCH"}),
			handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
			handlers.MaxAge(3600),
		)(r),
	)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "bridge says pong!",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
