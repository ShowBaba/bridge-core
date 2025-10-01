package endpoint

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

type Repository interface {
	get(ctx context.Context, q Endpoint) (*Endpoint, bool, error)
	getByAppMethodPath(ctx context.Context, appID, method, path string) (*Endpoint, bool, error)
	listByAppAndMethod(ctx context.Context, appID, method string) ([]Endpoint, error)
	getByAppMethodPathVersion(ctx context.Context, appID, method, path, version string) (*Endpoint, bool, error)
	listByAppMethodAndVersion(ctx context.Context, appID, method, version string) ([]Endpoint, error)
	delete(ctx context.Context, condition *Endpoint) error

	create(ctx context.Context, e *Endpoint) error
	update(ctx context.Context, e *Endpoint, updates Endpoint) error
	list(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error)
	deleteMany(ctx context.Context, ids []string) error
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db} }

func (r *repository) get(ctx context.Context, q Endpoint) (*Endpoint, bool, error) {
	var e Endpoint
	if err := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &e, true, nil
}

func (r *repository) getByAppMethodPath(ctx context.Context, appID, method, path string) (*Endpoint, bool, error) {
	var e Endpoint
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND method = ? AND path = ? AND deleted_at IS NULL", appID, method, path).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &e, true, nil
}

func (r *repository) listByAppAndMethod(ctx context.Context, appID, method string) ([]Endpoint, error) {
	var out []Endpoint
	if err := r.db.WithContext(ctx).
		Where("application_id = ? AND method = ? AND deleted_at IS NULL", appID, method).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) create(ctx context.Context, e *Endpoint) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *repository) update(ctx context.Context, e *Endpoint, updates Endpoint) error {
	return r.db.WithContext(ctx).Model(e).Where("id = ?", e.ID).Updates(updates).Error
}

func (r *repository) list(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error) {
	var out []Endpoint
	q := r.db.WithContext(ctx).Model(&Endpoint{}).Where(&filter).Where("deleted_at IS NULL")
	if opts.OrderBy != "" {
		dir := "ASC"
		if opts.OrderDesc {
			dir = "DESC"
		}
		q = q.Order(fmt.Sprintf("%s %s", opts.OrderBy, dir))
	}
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset)
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) deleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&Endpoint{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) delete(ctx context.Context, condition *Endpoint) error {
	return r.db.WithContext(ctx).Delete(condition).Error
}

func (r *repository) getByAppMethodPathVersion(ctx context.Context, appID, method, path, version string) (*Endpoint, bool, error) {
	var e Endpoint
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND method = ? AND path = ? AND version = ? AND deleted_at IS NULL",
			appID, method, path, version).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &e, true, nil
}

func (r *repository) listByAppMethodAndVersion(ctx context.Context, appID, method, version string) ([]Endpoint, error) {
	var out []Endpoint
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND method = ? AND version = ? AND deleted_at IS NULL",
			appID, method, version).
		Find(&out).Error
	return out, err
}
