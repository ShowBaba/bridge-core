package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Schema struct {
	ID         uint `gorm:"primaryKey"`
	DatabaseID uint `json:"database_id"`
	Name       string
	UserID     uint      `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Schema) Insert(db *gorm.DB) (uint, error) {
	if err := db.Create(s).Error; err != nil {
		return 0, err
	}
	return s.ID, nil
}

func (s *Schema) FetchSchemaByNameAndDatabaseID(db *gorm.DB) (*Schema, bool, error) {
	var schema Schema
	if err := db.Where("name = ? AND database_id = ?", s.Name, s.DatabaseID).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &schema, true, nil
}

func (s *Schema) Update(db *gorm.DB, updates Schema) error {
	return db.Model(s).Where("id = ?", s.ID).Updates(updates).Error
}

func (s *Schema) FetchSchema(db *gorm.DB, q Schema) (*Schema, bool, error) {
	var schema Schema
	if err := db.Where(q).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &schema, true, nil
}

func (s *Schema) DeleteMany(db *gorm.DB, ids []uint) error {
	result := db.Where("id IN ?", ids).Delete(&Schema{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Schema) FetchSchemas(db *gorm.DB, q Schema) ([]Schema, error) {
	var schemas []Schema
	if err := db.Where(q).Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}
