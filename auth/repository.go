package auth

import (
	"context"
	"errors"

	"github.com/showbaba/query-bridge/bridge-core/user"
	"gorm.io/gorm"
)

type Repository interface {
	getUser(ctx context.Context, q user.User) (*user.User, bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) getUser(ctx context.Context, q user.User) (*user.User, bool, error) {
	var u user.User
	if err := r.db.WithContext(ctx).
		Where(&q).
		Where("deleted_at IS NULL").
		First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &u, true, nil
}
