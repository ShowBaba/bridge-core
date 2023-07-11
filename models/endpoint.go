package models

import "time"

type Endpoint struct {
	ID            uint `gorm:"primaryKey"`
	ApplicationID uint `json:"application_id"`
	TableID       uint `json:"table_id"`
	Query         string
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
