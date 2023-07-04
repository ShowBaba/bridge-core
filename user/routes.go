package user

import (
	"database/sql"

	"github.com/gorilla/mux"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	db          *sql.DB
	messageChan *amqp.Channel
)

func InitializeUserRoutes(router *mux.Router, dbClient *sql.DB, channel *amqp.Channel) {
	messageChan = channel
	db = dbClient
	router.HandleFunc("/register", RegisterHandler).Methods("POST")
}
