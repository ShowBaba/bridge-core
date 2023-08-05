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
	UserID    uint      `json:"user_id"`
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

func (c *Column) FetchColumns(db *gorm.DB, q Column) ([]Column, error) {
	var columns []Column
	if err := db.Where(q).Find(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// delete many by ids
func (c *Column) DeleteMany(db *gorm.DB, ids []uint) error {
	result := db.Where("id IN ?", ids).Delete(&Column{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
