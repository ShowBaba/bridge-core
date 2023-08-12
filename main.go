package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

var (
	schema graphql.Schema
	ctx    = context.Background()
)

func main() {
	var (
		wg    sync.WaitGroup
		errCh = make(chan error)
	)
	qConn, err := amqp091.Dial(utils.GetConfig().RabbitmqServerURL)
	if err != nil {
		failOnError(err, "error creating queue connection", errCh)
	}
	defer qConn.Close()

	dbCl, pgConn, err := db.ConnectToPgDB(
		utils.GetConfig().DbHost,
		utils.GetConfig().DbUser,
		utils.GetConfig().DbPassword,
		utils.GetConfig().DbName,
		utils.GetConfig().DbPort,
	)
	if err != nil {
		failOnError(err, "error creating pg database connection", errCh)
	}
	defer pgConn.Close()

	mongoClient, ctx, cancel, err := db.ConnectToMongoDB(utils.GetConfig().MongoURI)
	if err != nil {
		failOnError(err, "error creating mongo database connection", errCh)
	}
	defer db.CloseDBConnection(mongoClient, ctx, cancel)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := database.InitDBQueue(dbCl, mongoClient, qConn); err != nil {
			failOnError(err, "eroor initializing database queue", errCh)
		}
	}()
	go func() {
		defer wg.Done()
		if err := notification.InitNotificationQueue(qConn); err != nil {
			failOnError(err, "err initilizing notification queue", errCh)
		}
	}()

	router := mux.NewRouter()
	db.Migrate(dbCl)
	InitializeRoutes(router, dbCl, qConn, mongoClient)
	// initialize graghql schema
	schema, err = graphql.NewSchema(
		graphql.SchemaConfig{
			Query: gql.Init(dbCl),
		},
	)
	if err != nil {
		fmt.Println("error creating schema: ", err)
		return
	}
	port := utils.GetConfig().Port
	if port == "" {
		port = "8080"
	}
	log.Printf("starting server on port: %s", port)
	if err := Run(router, fmt.Sprintf(`:%s`, port)); err != nil {
		failOnError(err, "fail to start server", errCh)
	}

	wg.Wait()
	close(errCh)

	go func() {
		for err := range errCh {
			log.Fatal(err)
		}
	}()

	// wait for a termination signal to gracefully shut down the server and the queues
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
		if err := qConn.Close(); err != nil {
			log.Printf("Error closing queue connection: %v", err)
		} else {
			log.Println("Queue connection closed.")
		}

		if err := pgConn.Close(); err != nil {
			log.Printf("Error closing pg connection: %v", err)
		} else {
			log.Println("PostgreSQL connection closed.")
		}

		if err := db.CloseDBConnection(mongoClient, ctx, cancel); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		} else {
			log.Println("MongoDB connection closed.")
		}
	default:
		log.Fatal("some unknown error occurred during shutdown")
	}

	log.Println("Server and database connections closed. Goodbye!")
}

func InitializeRoutes(router *mux.Router, dbCl *gorm.DB, qConnection *amqp091.Connection,
	mongoClient *mongo.Client) {
	// graphql route
	router.HandleFunc("/gql", func(w http.ResponseWriter, r *http.Request) {
		gql.RunGQL(w, r, schema, ctx)
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

// run
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

func failOnError(err error, msg string, ch chan<- error) {
	if err != nil {
		ch <- fmt.Errorf("%s: %s", msg, err)
	}
}
