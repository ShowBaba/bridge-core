package endpoint

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Endpoint struct {
	ID            string `gorm:"primaryKey"`
	Name          string
	ApplicationID string `json:"application_id"`
	TableID       string `json:"table_id"`
	DatabaseID    string `json:"database_id"`
	UserID        string `json:"user_id"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	Version       string `json:"version"`

	QueryTemplate  string         `json:"query_template"`
	IsPublic       *bool          `json:"is_public"`
	Columns        pq.StringArray `gorm:"type:varchar[]" json:"columns"`
	LimitDefault   *uint          `json:"limit_default"`
	LimitMax       *uint          `json:"limit_max"`
	OrderBy        string         `json:"order_by"`
	OrderDirection string         `json:"order_direction"`
	TimeoutMS      *int           `json:"timeout_ms"`
	URL            string         `json:"url" gorm:"-"`
	ParamSchema    datatypes.JSON `json:"param_schema" gorm:"type:jsonb"`
	QuerySchema    datatypes.JSON `json:"query_schema" gorm:"type:jsonb"`
	BodySchema     datatypes.JSON `json:"body_schema"  gorm:"type:jsonb"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (e *Endpoint) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	return nil
}

func (e *Endpoint) BeforeUpdate(tx *gorm.DB) (err error) {
	e.UpdatedAt = time.Now().UTC()
	return nil
}
