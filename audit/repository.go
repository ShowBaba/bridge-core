package audit

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	create(ctx context.Context, a *Audit) (*Audit, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) create(ctx context.Context, a *Audit) (*Audit, error) {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}
