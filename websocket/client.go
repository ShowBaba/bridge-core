package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func StreamHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, mongoClient *mongo.Client) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}
	if conn != nil {
		log.Println("new socket connection")
	}
	defer conn.Close()

	collection := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("error reading message:", err)
			return
		}
		var jsonInput wsInput
		if err = json.Unmarshal(message, &jsonInput); err != nil {
			log.Println("error unmershaling:", err)
			return
		}

		// valiadte token
		claim, err := utils.ValidateAuthToken(jsonInput.Token, utils.GetConfig().JWTSecretKey)
		if err != nil {
			log.Printf("error validating auth token token: %v", err)
			return
		}

		if jsonInput.StartTime.IsZero() {
			thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
			jsonInput.StartTime = thirtyMinutesAgo
		}

		if jsonInput.EndTime.IsZero() {
			jsonInput.EndTime = time.Now()
		}

		// Cursor to stream data from MongoDB collection
		cursor, err := collection.Find(ctx, bson.M{
			"application": jsonInput.ApplicationID,
			"user":        claim.ID,
			"created_at": bson.M{
				"$gte": jsonInput.StartTime,
				"$lt":  jsonInput.EndTime,
			},
		})
		if err != nil {
			log.Println("MongoDB query error:", err)
			return
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var logObj models.StreamLog
			err := cursor.Decode(&logObj)
			if err != nil {
				log.Println("Cursor decoding error:", err)
				return
			}

			fmt.Println(logObj)

			// Send log object as JSON to the WebSocket client
			err = conn.WriteJSON(logObj)
			if err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}

		if err := cursor.Err(); err != nil {
			log.Println("Cursor error:", err)
			return
		}
	}
}
