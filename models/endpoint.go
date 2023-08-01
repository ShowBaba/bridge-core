package models

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Endpoint struct {
	ID             uint `gorm:"primaryKey"`
	Name           string
	ApplicationID  uint `json:"application_id"`
	TableID        uint `json:"table_id"`
	Query          string
	UserID         uint           `json:"user_id"`
	IdentifierUUID string         `json:"identifier_uuid"`
	IsPublic       *bool          `json:"is_public"`
	Method         string         `json:"method"`
	Columns        pq.StringArray `gorm:"type:varchar[]"`
	Limit          uint           `json:"limit"`
	OrderBy        string         `json:"order_by"`
	OrderDirection string         `json:"order_direction"`
	DatabaseID uint `json:"database_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (e *Endpoint) Insert(db *gorm.DB) error {
	if err := db.Create(e).Error; err != nil {
		return err
	}
	return nil
}

func (e *Endpoint) FetchEndpoint(db *gorm.DB, q Endpoint) (*Endpoint, bool, error) {
	var endpoint Endpoint
	if err := db.Where(q).First(&endpoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &endpoint, true, nil
}

func (e *Endpoint) Update(db *gorm.DB, updates Endpoint) error {
	if err := db.Model(e).Where("id = ?", e.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func (e *Endpoint) Delete(db *gorm.DB, condition *Endpoint) error {
	result := db.Where(&condition).Delete(e)
	return result.Error
}

func (e *Endpoint) FetchEndpoints(db *gorm.DB, q Endpoint) ([]Endpoint, error) {
	var endpoints []Endpoint
	if err := db.Where(q).Find(&endpoints).Error; err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (e *Endpoint) DeleteMany(db *gorm.DB, ids []uint) error {
	result := db.Where("id IN ?", ids).Delete(&Endpoint{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
