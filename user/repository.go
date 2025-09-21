package user

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	get(ctx context.Context, q User) (*User, bool, error)
	create(ctx context.Context, u *User) (*User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) get(ctx context.Context, q User) (*User, bool, error) {
	var u User
	if err := r.db.WithContext(ctx).Where(&q).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &u, true, nil
}

func (r *repository) create(ctx context.Context, u *User) (*User, error) {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}
