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
	Slug      string         `json:"slug"`
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

type ApplicationEnv struct {
	ID            string         `gorm:"primaryKey;size:32" json:"id"`
	ApplicationId string         `json:"application_id"`
	Key           string         `gorm:"null"`
	Value         string         `gorm:"not null"`
	Status        string         `gorm:"default:active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"-"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *ApplicationEnv) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *ApplicationEnv) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

type EndpointScript struct {
	ID         string `gorm:"primaryKey;size:32"`
	EndpointID string `gorm:"index"`
	Kind       string `gorm:"size:8"` // "pre" | "post"
	Lang       string `gorm:"size:8"` // "js"
	Code       string `gorm:"type:text"`
	Enabled    bool
	CreatedBy  string
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *EndpointScript) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *EndpointScript) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}
