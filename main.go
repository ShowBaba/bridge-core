package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/auth"
	"github.com/showbaba/query-bridge/bridge/notification"
	"github.com/showbaba/query-bridge/bridge/db"
	"github.com/showbaba/query-bridge/bridge/user"
	"github.com/showbaba/query-bridge/bridge/utils"
	"github.com/showbaba/query-bridge/shared"
)

func main() {
	connection, err := amqp.Dial(utils.GetConfig().RabbitmqServerURL)
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	go notification.InitializeNotificationQueue(connection)
	messageChan, err := connection.Channel()
	if err != nil {
		panic(err)
	}
	defer messageChan.Close()
	dbCl := shared.ConnectToSQLDB(
		utils.GetConfig().DbHost,
		utils.GetConfig().DbUser,
		utils.GetConfig().DbPassword,
		utils.GetConfig().DbName,
		utils.GetConfig().DbPort,
	)
	defer dbCl.Close()

	router := mux.NewRouter()
	db.Migrate(dbCl)
	InitializeRoutes(router, dbCl, messageChan)
	port := utils.GetConfig().Port
	log.Printf("starting server on port: %s", port)
	Run(router, port)
}

func InitializeRoutes(router *mux.Router, dbCl *sql.DB, channel *amqp.Channel) {
	router.HandleFunc("/ping", ping).Methods("GET")
	authRoutes := router.PathPrefix("/auth").Subrouter()
	auth.InitializeAuthRoutes(authRoutes, dbCl)

	userRoutes := router.PathPrefix("/user").Subrouter()
	user.InitializeUserRoutes(userRoutes, dbCl, channel)

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
	response := shared.APIResponse{
		Status:  http.StatusOK,
		Message: "bridge says pong!",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	w.Write(responseJSON)
}
