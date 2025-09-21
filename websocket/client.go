package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	log2 "github.com/showbaba/query-bridge/bridge-core/database-log"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func StreamHandler(ctx context.Context, mongoClient *mongo.Client) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		defer c.Close()

		log.Println("new socket connection")

		collection := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).
			Collection(utils.QUERY_BRIDGE_MONGO_LOGS_COLLECTION)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("error reading message:", err)
				return
			}

			var jsonInput wsInput
			if err = json.Unmarshal(message, &jsonInput); err != nil {
				log.Println("error unmarshaling:", err)
				return
			}

			claim, err := utils.ValidateAuthToken(jsonInput.Token, utils.GetConfig().JWTSecretKey)
			if err != nil {
				log.Printf("error validating auth token: %v", err)
				return
			}

			if jsonInput.StartTime.IsZero() {
				jsonInput.StartTime = time.Now().Add(-30 * time.Minute)
			}
			if jsonInput.EndTime.IsZero() {
				jsonInput.EndTime = time.Now()
			}

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
				var logObj log2.StreamLog
				if err := cursor.Decode(&logObj); err != nil {
					log.Println("Cursor decoding error:", err)
					return
				}

				fmt.Println(logObj)

				if err := c.WriteJSON(logObj); err != nil {
					log.Println("WebSocket write error:", err)
					return
				}
			}

			if err := cursor.Err(); err != nil {
				log.Println("Cursor error:", err)
				return
			}
		}
	})
}
