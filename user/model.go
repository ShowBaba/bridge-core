package user

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string `gorm:"primaryKey"`
	Email     string
	FirstName string
	LastName  string
	Password  string
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UpdatedAt = time.Now().Local()
	u.CreatedAt = time.Now().Local()
	u.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
	return
}
