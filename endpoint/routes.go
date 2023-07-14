package endpoint

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/showbaba/query-bridge/bridge/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

var (
	db              *gorm.DB
	ctx             = context.Background()
	mongoClient     *mongo.Client
)

func InitializeEndpointRoutes(router *mux.Router, dbClient *gorm.DB, mongoCl *mongo.Client) {
	db = dbClient
	mongoClient = mongoCl
	router.HandleFunc("/create", utils.ValidateAuthHeaderToken(http.HandlerFunc(CreateEndpointHandler))).Methods("POST")
	router.HandleFunc("/execute/{identifier}", utils.ValidateAuthHeaderToken(http.HandlerFunc(ExecuteEndpointHandler))).Methods("POST")
	router.HandleFunc("/update/{endpoint_id}", utils.ValidateAuthHeaderToken(http.HandlerFunc(UpdateEndpointHandler))).Methods("PATCH")
}
