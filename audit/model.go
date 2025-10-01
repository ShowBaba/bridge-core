package audit

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Audit struct {
	ID            string         `gorm:"primaryKey;size:32" json:"id"`
	UserID        string         `json:"user_id"`
	ApplicationID string         `json:"application_id"`
	Action        string         `json:"action"`
	EntityType    string         `json:"entity_type"`
	EntityID      string         `json:"entity_id"`
	Description   string         `json:"description,omitempty"`
	Metadata      string         `json:"metadata,omitempty"`
	IPAddress     string         `json:"ip_address,omitempty"`
	Username      string         `gorm:"-" json:"username,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"-"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Audit) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Audit) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}
