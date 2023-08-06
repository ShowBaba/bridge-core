package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

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

	log.Println("pg database connection established!")
	return db, sqlDB, nil
}

func ConnectToMongoDB(uri string) (*mongo.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, ctx, cancel, err
	}
	pingMongoDB(ctx, client)
	return client, ctx, cancel, err
}

func pingMongoDB(ctx context.Context, client *mongo.Client) {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}
	log.Println("mongo database connection established!")
}

func CloseDBConnection(client *mongo.Client, ctx context.Context,
	cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}