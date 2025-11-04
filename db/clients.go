package db

import (
	"context"
	"database/sql"
	"fmt"

	log "github.com/showbaba/query-bridge/bridge-core/logger"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectToPgDB(host, user, password, dbname string, port int) (*gorm.DB, *sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, nil, err
	}

	log.Info("pg database connection established!")
	return db, sqlDB, nil
}

func ConnectToMongoDB(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	pingMongoDB(ctx, client)
	return client, err
}

func EnsureMongoIndexes(ctx context.Context, mongoClient *mongo.Client) error {
	coll := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user", Value: 1}}},
		{Keys: bson.D{{Key: "application", Value: 1}}},
		{Keys: bson.D{{Key: "level", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	_, err := coll.Indexes().CreateMany(ctx, indexes)
	return err
}

func pingMongoDB(ctx context.Context, client *mongo.Client) {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}
	log.Info("mongo database connection established!")
}

func CloseDBConnection(client *mongo.Client, ctx context.Context) error {
	return client.Disconnect(ctx)
}
