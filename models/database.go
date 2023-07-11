package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Database struct {
	ID            uint `gorm:"primaryKey"`
	Name          string
	Host          string
	Port          uint
	Database      string
	Username      string
	Password      string
	DbEngine      string    `json:"db_engine"`
	ApplicationID uint      `json:"database_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (d *Database) Insert(db *gorm.DB) (uint, error) {
	if err := db.Create(d).Error; err != nil {
		return 0, err
	}
	return d.ID, nil
}

func (d *Database) FetchDatabase(db *gorm.DB, q Database) (*Database, bool, error) {
	var database Database
	if err := db.Where(q).First(&database).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &database, true, nil
}

func (d *Database) Update(db *gorm.DB, updates map[string]interface{}) error {
	if err := db.Model(d).Where("id = ?", d.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (d *Database) Delete(db *gorm.DB) error {
	if err := db.Delete(d).Error; err != nil {
		return err
	}
	return nil
}





