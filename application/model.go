package application

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Application struct {
	ID        string         `gorm:"primaryKey;size:32" json:"id"`
	Name      string         `json:"name"`
	UserID    string         `json:"user_id"`
	ApiKey    string         `json:"api_key"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Application) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}
