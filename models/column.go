package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Column struct {
	ID        uint `gorm:"primaryKey"`
	TableID   uint `json:"table_id"`
	Name      string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Column) Insert(db *gorm.DB) error {
	if err := db.Create(c).Error; err != nil {
		return err
	}
	return nil
}

func (c *Column) FetchColumnByNameAndTableID(db *gorm.DB) (*Column, bool, error) {
	var column Column
	if err := db.Where("name = ? AND table_id = ?", c.Name, c.TableID).First(&column).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &column, true, nil
}

func (c *Column) Update(db *gorm.DB, updates Column) error {
	if err := db.Model(c).Where("id = ?", c.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}
