package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

type Repository interface {
	create(ctx context.Context, a *Database) (*Database, error)
	get(ctx context.Context, q Database) (*Database, bool, error)
	getByID(ctx context.Context, id string) (*Database, bool, error)
	getByDbHostApp(ctx context.Context, dbName, host, appID string) (*Database, bool, error)
	List(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error)
	count(ctx context.Context, filter Database) (int64, error)
	update(ctx context.Context, d *Database, updates map[string]interface{}) error
	delete(ctx context.Context, condition *Database) error
	restore(ctx context.Context, id string) error
	deleteMany(ctx context.Context, ids []string) error
	list(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error)

	getTable(ctx context.Context, q Table) (*Table, bool, error)
	listTables(ctx context.Context, q Table) ([]Table, error)
	listTablesByDatabase(ctx context.Context, databaseID string, opts utils.ListOpts) ([]Table, error)
	updateTable(ctx context.Context, t *Table, updates map[string]interface{}) error
	deleteTable(ctx context.Context, cond *Table) error
	restoreTable(ctx context.Context, id string) error
	deleteManyTables(ctx context.Context, ids []string) error

	listColumns(ctx context.Context, q Column) ([]Column, error)
	deleteManyColumns(ctx context.Context, ids []string) error

	getSchema(ctx context.Context, q Schema) (*Schema, bool, error)
	listSchemas(ctx context.Context, q Schema) ([]Schema, error)
	restoreSchema(ctx context.Context, id string) error
	deleteManySchemas(ctx context.Context, ids []string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) create(ctx context.Context, a *Database) (*Database, error) {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (r *repository) get(ctx context.Context, q Database) (*Database, bool, error) {
	var d Database
	tx := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").First(&d)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &d, true, nil
}

func (r *repository) getByID(ctx context.Context, id string) (*Database, bool, error) {
	var d Database
	tx := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&d)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &d, true, nil
}

func (r *repository) getByDbHostApp(ctx context.Context, dbName, host, appID string) (*Database, bool, error) {
	var d Database
	tx := r.db.WithContext(ctx).
		Where("database = ? AND host = ? AND application_id = ? AND deleted_at IS NULL", dbName, host, appID).
		First(&d)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &d, true, nil
}

func (r *repository) List(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error) {
	var out []Database
	q := r.db.WithContext(ctx).Model(&Database{}).Where(&filter).Where("deleted_at IS NULL")
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

func (r *repository) count(ctx context.Context, filter Database) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).
		Model(&Database{}).
		Where(&filter).
		Where("deleted_at IS NULL").
		Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func (r *repository) update(ctx context.Context, d *Database, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&Database{}).
		Where("id = ? AND deleted_at IS NULL", d.ID).
		Updates(updates).Error
}

func (r *repository) delete(ctx context.Context, condition *Database) error {
	now := time.Now().UTC()
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Table("endpoints").
		Where("database_id = ? AND deleted_at IS NULL", condition.ID).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&Column{}).
		Where("deleted_at IS NULL").
		Where("table_id IN (?)",
			tx.Model(&Table{}).Select("id").
				Where("deleted_at IS NULL").
				Where("schema_id IN (?)",
					tx.Model(&Schema{}).Select("id").
						Where("database_id = ? AND deleted_at IS NULL", condition.ID),
				),
		).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&Table{}).
		Where("deleted_at IS NULL").
		Where("schema_id IN (?)",
			tx.Model(&Schema{}).Select("id").
				Where("database_id = ? AND deleted_at IS NULL", condition.ID),
		).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&Schema{}).
		Where("database_id = ? AND deleted_at IS NULL", condition.ID).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&Database{}).
		Where("id = ? AND deleted_at IS NULL", condition.ID).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *repository) restore(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&Database{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil).Error
}

func (r *repository) getTable(ctx context.Context, q Table) (*Table, bool, error) {
	var t Table
	tx := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").First(&t)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &t, true, nil
}

func (r *repository) listTablesByDatabase(ctx context.Context, databaseID string, opts utils.ListOpts) ([]Table, error) {
	var out []Table
	q := r.db.WithContext(ctx).Model(&Table{}).Where("database_id = ? AND deleted_at IS NULL", databaseID)
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

func (r *repository) updateTable(ctx context.Context, t *Table, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&Table{}).
		Where("id = ? AND deleted_at IS NULL", t.ID).
		Updates(updates).Error
}

func (r *repository) deleteTable(ctx context.Context, cond *Table) error {
	return r.db.WithContext(ctx).
		Model(&Table{}).
		Where("id = ? AND deleted_at IS NULL", cond.ID).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) restoreTable(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&Table{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil).Error
}

func (r *repository) listColumns(ctx context.Context, q Column) ([]Column, error) {
	var cols []Column
	if err := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").Find(&cols).Error; err != nil {
		return nil, err
	}
	return cols, nil
}

func (r *repository) listSchemas(ctx context.Context, q Schema) ([]Schema, error) {
	var schemas []Schema
	if err := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}

func (r *repository) listTables(ctx context.Context, q Table) ([]Table, error) {
	var tables []Table
	if err := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").Find(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

func (r *repository) getSchema(ctx context.Context, q Schema) (*Schema, bool, error) {
	var s Schema
	tx := r.db.WithContext(ctx).Where(&q).Where("deleted_at IS NULL").First(&s)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &s, true, nil
}

func (r *repository) restoreSchema(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&Schema{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil).Error
}

func (r *repository) deleteManyColumns(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&Column{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) deleteManyTables(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&Table{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) deleteManySchemas(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&Schema{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) deleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&Database{}).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Update("deleted_at", time.Now().UTC()).Error
}

func (r *repository) list(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error) {
	var out []Database
	q := r.db.WithContext(ctx).Model(&Database{}).Where(&filter).Where("deleted_at IS NULL")
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
