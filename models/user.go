package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primaryKey"`
	Email     string
	FirstName string
	LastName  string
	Password  string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) Insert(db *gorm.DB) (uint, error) {
	if err := db.Create(u).Error; err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (u *User) Update(db *gorm.DB) (*User, error) {
	return nil, nil
}

func (u *User) Delete(db *gorm.DB) error {
	return nil
}

func (u *User) DeleteByID(db *gorm.DB, id int) error {
	return nil
}

func (u *User) GetAll(db *gorm.DB) ([]*User, error) {
	return nil, nil
}

func (u *User) GetUser(db *gorm.DB, q User) (*User, error) {
	var user User
	err := db.Where(q).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *User) GetByID(idb *gorm.DB, d int) (*User, error) {
	return nil, nil
}
