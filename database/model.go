package database

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Database struct {
	ID            string `gorm:"primaryKey"`
	Name          string
	Host          string
	Port          uint
	Database      string
	Username      string
	Password      string
	DbEngine      string         `json:"db_engine"`
	ApplicationID string         `json:"application_id"`
	UserID        string         `json:"user_id"`
	SSLMode       string         `json:"ssl_mode"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Database) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Database) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

type Table struct {
	ID         string `gorm:"primaryKey"`
	SchemaID   string `json:"schema_id"`
	DatabaseID string `json:"database_id"`
	UserID     string `json:"user_id"`
	Name       string
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	Indexes    []Index        `gorm:"foreignKey:TableID"`
}

func (a *Table) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Table) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

type Column struct {
	ID           string         `gorm:"primaryKey" json:"id"`
	TableID      string         `json:"table_id"`
	Name         string         `json:"name"`
	DataType     string         `json:"data_type"`
	IsNullable   bool           `json:"is_nullable"`
	DefaultValue sql.NullString `json:"default_value"`
	IsPrimaryKey bool           `json:"is_primary_key"`
	FKRefSchema  sql.NullString `json:"fk_ref_schema"`
	FKRefTable   sql.NullString `json:"fk_ref_table"`
	FKRefColumn  sql.NullString `json:"fk_ref_column"`
	FKConstraint sql.NullString `json:"fk_constraint"`
	UserID       string         `json:"user_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Column) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Column) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

type Schema struct {
	ID         string `gorm:"primaryKey"`
	DatabaseID string `json:"database_id"`
	Name       string
	UserID     string         `json:"user_id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Schema) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Schema) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

type Index struct {
	ID         string `gorm:"primaryKey"`
	TableID    string `json:"table_id"`
	UserID     string `json:"user_id"`
	Name       string
	Definition string `gorm:"type:text"`
	IsUnique   bool
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Index) TableName() string {
	return "indexes"
}

func (a *Index) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return nil
}

func (a *Index) BeforeUpdate(tx *gorm.DB) (err error) {
	a.UpdatedAt = time.Now().UTC()
	return nil
}
