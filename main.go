package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/graphql-go/graphql"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/application"
	"github.com/showbaba/query-bridge/bridge/auth"
	"github.com/showbaba/query-bridge/bridge/database"
	"github.com/showbaba/query-bridge/bridge/db"
	"github.com/showbaba/query-bridge/bridge/gql"
	"github.com/showbaba/query-bridge/bridge/notification"
	"github.com/showbaba/query-bridge/bridge/user"
	"github.com/showbaba/query-bridge/bridge/utils"

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
	InitializeRoutes(router, dbCl, qConn)
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

func InitializeRoutes(router *mux.Router, dbCl *gorm.DB, qConnection *amqp091.Connection) {
	// graphql route
	router.HandleFunc("/gql", runGQL).Methods("POST", "OPTIONS")
	router.HandleFunc("/ping", ping).Methods("GET")

	authRoutes := router.PathPrefix("/auth").Subrouter()
	auth.InitializeAuthRoutes(authRoutes, dbCl)

	userRoutes := router.PathPrefix("/user").Subrouter()
	user.InitializeUserRoutes(userRoutes, dbCl, qConnection)

	applicationRoutes := router.PathPrefix("/application").Subrouter()
	application.InitializeApplicationRoutes(applicationRoutes, dbCl, qConnection)

	databaseRoutes := router.PathPrefix("/database").Subrouter()
	database.InitializeApplicationRoutes(databaseRoutes, dbCl, qConnection)
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

func runGQL(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Add("Access-Control-Allow-Headers", "Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Read the query
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		utils.Dispatch400Error(w, "invalid request body: %s")
		return
	}

	var (
		payload gql.GraphQLPayload
		resp    *graphql.Result
	)

	if err := json.Unmarshal(body, &payload); err == nil {
		// Perform GraphQL request
		resp = graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  payload.Query,
			VariableValues: payload.Variables,
			Context:        ctx,
		})
	} else {
		resp = graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: string(body),
			Context:       ctx,
		})
	}
	if len(resp.Errors) > 0 {
		utils.Dispatch400Error(w, fmt.Sprintf("%+v", resp.Errors))
		return
	}
	responseJSON(w, resp)
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
