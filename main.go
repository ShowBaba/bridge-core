package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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
	qConn, err := amqp091.Dial(utils.GetConfig().RabbitmqServerURL)
	if err != nil {
		panic(err)
	}
	defer qConn.Close()

	dbCl, conn, err := db.ConnectToPgDB(
		utils.GetConfig().DbHost,
		utils.GetConfig().DbUser,
		utils.GetConfig().DbPassword,
		utils.GetConfig().DbName,
		utils.GetConfig().DbPort,
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	mongoClient, ctx, cancel, err := db.ConnectToMongoDB(utils.GetConfig().MongoURI)
	if err != nil {
		panic(err)
	}
	defer db.CloseDBConnection(mongoClient, ctx, cancel)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		database.InitDBQueue(dbCl, mongoClient, qConn)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		notification.InitNotificationQueue(qConn)
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
	log.Printf("starting server on port: %s", port)
	Run(router, port)
	wg.Wait()
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
func Run(r *mux.Router, host string) {
	// CORS
	log.Fatal(
		http.ListenAndServe(
			host,
			handlers.CORS(
				handlers.AllowCredentials(),
				handlers.AllowedMethods([]string{"POST", "GET", "PUT", "OPTIONS"}),
				handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
				handlers.MaxAge(3600),
			)(r),
		),
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
