package endpoint

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

var (
	db          *gorm.DB
	ctx         = context.Background()
	mongoClient *mongo.Client
)

func InitializeEndpointRoutes(router *mux.Router, dbClient *gorm.DB, mongoCl *mongo.Client) {
	db = dbClient
	mongoClient = mongoCl
	router.HandleFunc("/{database_id}/create", utils.ValidateAuthHeaderToken(CreateEndpointHandler)).Methods("POST")
	router.HandleFunc("/execute/{identifier}", utils.ValidateAuthHeaderToken(ExecuteEndpointHandler)).Methods("POST")
	router.HandleFunc("/update/{endpoint_id}", utils.ValidateAuthHeaderToken(UpdateEndpointHandler)).Methods("PATCH")
}
