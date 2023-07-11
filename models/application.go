package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Application struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	UserID    uint `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *Application) Insert(db *gorm.DB) error {
	if err := db.Create(a).Error; err != nil {
		return err
	}
	return nil
}

func (a *Application) FetchApplication(db *gorm.DB, q Application) (*Application, error) {
	var application Application
	if err := db.Where(q).First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &application, nil
}
