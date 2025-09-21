package database_log

import (
	"context"

	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repository interface {
	Create(ctx context.Context, data StreamLog) error
}

type repository struct {
	mongoClient *mongo.Client
}

func NewRepository(mongoClient *mongo.Client) Repository {
	return &repository{mongoClient}
}

func (r *repository) Create(ctx context.Context, data StreamLog) error {
	collection := r.mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)
	_, err := collection.InsertOne(ctx, data)
	return err
}
