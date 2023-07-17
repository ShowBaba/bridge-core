package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Application struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	UserID    uint      `json:"user_id"`
	ApiKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *Application) Insert(db *gorm.DB) error {
	return db.Create(a).Error
}

func (a *Application) FetchApplication(db *gorm.DB, q Application) (*Application, bool, error) {
	var application Application
	if err := db.Where(q).First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &application, true, nil
}

func (a *Application) Update(db *gorm.DB, updates Application) error {
	return db.Model(a).Where("id = ?", a.ID).Updates(updates).Error
}

func (a *Application) Delete(db *gorm.DB, condition *Application) error {
	result := db.Where(&condition).Delete(a)
	return result.Error
}
