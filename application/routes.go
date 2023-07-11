package application

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge/utils"
	"gorm.io/gorm"
)

var (
	db      *gorm.DB
	ctx     = context.Background()
	queueConnection *amqp091.Connection
)

func InitializeApplicationRoutes(router *mux.Router, dbClient *gorm.DB, qC *amqp091.Connection) {
	db = dbClient
	queueConnection = qC
	router.HandleFunc("/create", utils.ValidateAuthToken(http.HandlerFunc(CreateApplication))).Methods("POST")
	router.HandleFunc("/{application_id}/add-database", utils.ValidateAuthToken(http.HandlerFunc(AddDatabases))).Methods("POST")
}
