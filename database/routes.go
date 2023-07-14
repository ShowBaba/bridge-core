package database

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/utils"
	"gorm.io/gorm"
)

var (
	db              *gorm.DB
	queueConnection *amqp091.Connection
)

func InitializeApplicationRoutes(router *mux.Router, dbClient *gorm.DB, qC *amqp091.Connection) {
	db = dbClient
	queueConnection = qC
	router.HandleFunc("/{database_id}", utils.ValidateAuthHeaderToken(http.HandlerFunc(UpdateDatabase))).Methods("PATCH")
}
