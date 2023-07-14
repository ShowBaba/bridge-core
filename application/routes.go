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
	db              *gorm.DB
	ctx             = context.Background()
	queueConnection *amqp091.Connection
)

func InitializeApplicationRoutes(router *mux.Router, dbClient *gorm.DB, qC *amqp091.Connection) {
	db = dbClient
	queueConnection = qC
	router.HandleFunc("/create", utils.ValidateAuthHeaderToken(http.HandlerFunc(CreateApplication))).Methods("POST")
	router.HandleFunc("/{application_id}/update", utils.ValidateAuthHeaderToken(http.HandlerFunc(UpdateApplication))).Methods("PATCH")
	router.HandleFunc("/{application_id}/add-database", utils.ValidateAuthHeaderToken(http.HandlerFunc(AddDatabases))).Methods("POST")
}
