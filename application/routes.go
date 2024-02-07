package application

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/utils"
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
	router.HandleFunc("/create", utils.ValidateAuthHeaderToken(CreateApplication)).Methods("POST")
	router.HandleFunc("/{application_id}/update", utils.ValidateAuthHeaderToken(UpdateApplication)).Methods("PATCH")
	router.HandleFunc("/{application_id}/delete", utils.ValidateAuthHeaderToken(DeleteApplication)).Methods("DELETE")
	router.HandleFunc("/{application_id}/add-database", utils.ValidateAuthHeaderToken(AddDatabases)).Methods("POST")
}
