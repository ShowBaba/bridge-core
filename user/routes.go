package user

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

var (
	db      *gorm.DB
	queueConnection *amqp091.Connection
	ctx     = context.Background()
)

func InitializeUserRoutes(router *mux.Router, dbClient *gorm.DB, qC *amqp091.Connection) {
	queueConnection = qC
	db = dbClient
	router.HandleFunc("/register", RegisterHandler).Methods("POST")
}
