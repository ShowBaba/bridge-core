package models

import "time"

type StreamLog struct {
	Message   string    `bson:"message" json:"message"`
	Level     string    `bson:"level" json:"level"`
	Source    string    `bson:"source" json:"source"`
	Timestamp  string    `bson:"timestamp" json:"timestamp"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

