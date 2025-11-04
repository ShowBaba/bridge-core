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

	create(ctx context.Context, e *Endpoint) (*Endpoint, error)
	update(ctx context.Context, e *Endpoint, updates Endpoint) error
	list(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error)
	deleteMany(ctx context.Context, ids []string) error

	upsertScripts(ctx context.Context, endpointID string, scripts []EndpointScript, userID string) error
	getScripts(ctx context.Context, endpointID string) ([]EndpointScript, error)
	deleteScriptsByEndpoint(ctx context.Context, endpointID string) error

	createScripts(ctx context.Context, scripts []EndpointScript) error
	listScriptsByEndpoint(ctx context.Context, endpointID string) ([]EndpointScript, error)
	createEndpointStat(ctx context.Context, e *EndpointStats) error
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

func (r *repository) create(ctx context.Context, a *Endpoint) (*Endpoint, error) {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (r *repository) createEndpointStat(ctx context.Context, e *EndpointStats) error {
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

func (r *repository) upsertScripts(ctx context.Context, endpointID string, scripts []EndpointScript, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("endpoint_id = ?", endpointID).
			Delete(&EndpointScript{}).Error; err != nil {
			return err
		}

		if len(scripts) == 0 {
			return nil
		}

		for i := range scripts {
			scripts[i].EndpointID = endpointID
			scripts[i].UserID = userID
		}
		if err := tx.Create(&scripts).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) getScripts(ctx context.Context, endpointID string) ([]EndpointScript, error) {
	var out []EndpointScript
	if err := r.db.WithContext(ctx).
		Where("endpoint_id = ? AND deleted_at IS NULL", endpointID).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) deleteScriptsByEndpoint(ctx context.Context, endpointID string) error {
	return r.db.WithContext(ctx).
		Where("endpoint_id = ? AND deleted_at IS NULL", endpointID).
		Delete(&EndpointScript{}).Error
}

func (r *repository) createScripts(ctx context.Context, scripts []EndpointScript) error {
	if len(scripts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&scripts).Error
}

func (r *repository) listScriptsByEndpoint(ctx context.Context, endpointID string) ([]EndpointScript, error) {
	var out []EndpointScript
	if err := r.db.WithContext(ctx).
		Where("endpoint_id = ? AND deleted_at IS NULL", endpointID).
		Order("created_at ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
