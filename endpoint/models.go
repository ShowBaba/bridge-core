package endpoint

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Endpoint struct {
	ID             string `gorm:"primaryKey"`
	Name           string
	ApplicationID  string `json:"application_id"`
	TableID        string `json:"table_id"`
	Query          string
	UserID         string         `json:"user_id"`
	IdentifierUUID string         `json:"identifier_uuid"`
	IsPublic       *bool          `json:"is_public"`
	Method         string         `json:"method"`
	Columns        pq.StringArray `gorm:"type:varchar[]"`
	Limit          uint           `json:"limit"`
	OrderBy        string         `json:"order_by"`
	OrderDirection string         `json:"order_direction"`
	DatabaseID     string         `json:"database_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Endpoint) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Endpoint) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}
