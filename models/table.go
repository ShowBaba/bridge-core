package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Table struct {
	ID        uint `gorm:"primaryKey"`
	SchemaID  uint `json:"schema_id"`
	Name      string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *Table) Insert(db *gorm.DB) (uint, error) {
	if err := db.Create(t).Error; err != nil {
		return 0, err
	}
	return t.ID, nil
}

func (t *Table) FetchTableByNameAndSchemaID(db *gorm.DB) (*Table, bool, error) {
	var table Table
	if err := db.Where("name = ? AND schema_id = ?", t.Name, t.SchemaID).First(&table).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &table, true, nil
}

func (t *Table) Update(db *gorm.DB, updates Table) error {
	if err := db.Model(t).Where("id = ?", t.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}
