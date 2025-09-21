package application

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	create(ctx context.Context, a *Application) error
	get(ctx context.Context, q Application) (*Application, bool, error)
	update(ctx context.Context, a *Application, updates Application) error
	delete(ctx context.Context, condition *Application) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db}
}

func (r *repository) create(ctx context.Context, a *Application) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *repository) get(ctx context.Context, q Application) (*Application, bool, error) {
	var application Application
	if err := r.db.WithContext(ctx).
		Where(&q).
		Where("deleted_at IS NULL").
		First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &application, true, nil
}

func (r *repository) update(ctx context.Context, a *Application, updates Application) error {
	set := map[string]interface{}{}
	if updates.Name != "" {
		set["name"] = updates.Name
	}
	if updates.ApiKey != "" {
		set["api_key"] = updates.ApiKey
	}
	if len(set) == 0 {
		return nil
	}
	set["updated_at"] = time.Now().UTC()

	return r.db.WithContext(ctx).
		Model(&Application{}).
		Where("id = ? AND deleted_at IS NULL", a.ID).
		Updates(set).Error
}

func (r *repository) delete(ctx context.Context, condition *Application) error {
	return r.db.WithContext(ctx).
		Model(&Application{}).
		Where("id = ? AND deleted_at IS NULL", condition.ID).
		Update("deleted_at", time.Now().UTC()).Error
}
