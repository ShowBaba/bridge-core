package models

import (
	"context"
	"time"

	"github.com/showbaba/query-bridge/bridge/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

type StreamLog struct {
	Message     string    `bson:"message" json:"message"`
	Level       string    `bson:"level" json:"level"`
	Source      string    `bson:"source" json:"source"`
	Timestamp   string    `bson:"timestamp" json:"timestamp"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
	Process     string    `bson:"process" json:"process"`
	Application uint      `bson:"application" json:"application"`
	User        uint      `bson:"user" json:"user"`
}

func (s *StreamLog) Insert(ctx context.Context, mongoClient *mongo.Client, logData string,
	applicationID, userID uint) error {
	collection := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)
	currentTime := time.Now()
	_, err := collection.InsertOne(ctx, StreamLog{
		Message:     logData,
		Source:      "DATABASE",
		Level:       "INFO",
		Timestamp:   currentTime.Format("2006-01-02 15:04:05"),
		CreatedAt:   currentTime,
		UpdatedAt:   currentTime,
		Process:     "DatabaseQueryProcess",
		Application: applicationID,
		User:        userID,
	})
	return err
}
