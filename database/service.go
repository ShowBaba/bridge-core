package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrDuplicateDBName = errors.New("duplicate database name")
)

type Service interface {
	List(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error)
	Get(ctx context.Context, application *Database) (*Database, error)
	GetTable(ctx context.Context, application *Table) (*Table, error)
	update(ctx context.Context, userID, databaseID string, p UpdateDatabasePayload) error
	delete(ctx context.Context, userID, databaseID string) error
	DeleteMany(ctx context.Context, ids []string) error
	add(ctx context.Context, userID, appID string, payload AddDatabasePayload) error
	ListColumns(ctx context.Context, q Column) ([]Column, error)
	GetSchema(ctx context.Context, q Schema) (*Schema, error)
	ListSchemas(ctx context.Context, q Schema) ([]Schema, error)
	DeleteManySchemas(ctx context.Context, ids []string) error
	DeleteManyColumns(ctx context.Context, ids []string) error
	ListTables(ctx context.Context, q Table) ([]Table, error)
	DeleteManyTables(ctx context.Context, ids []string) error
}

type service struct {
	repo           Repository
	applicationSvc application.Service
	auditSvc       audit.Service
	qConn          *amqp091.Connection
}

func NewService(repo Repository, applicationSvc application.Service, auditSvc audit.Service, q *amqp091.Connection) Service {
	return &service{repo: repo, applicationSvc: applicationSvc, auditSvc: auditSvc, qConn: q}
}

func (s *service) DeleteMany(ctx context.Context, ids []string) error {
	if err := s.repo.deleteMany(ctx, ids); err != nil {
		return err
	}
	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      "",
		Action:      "delete_many",
		EntityType:  "database",
		EntityID:    "",
		Description: fmt.Sprintf("soft-deleted %d databases", len(ids)),
		Metadata: map[string]any{
			"ids": ids,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: "",
	})
	return nil
}

func (s *service) DeleteManyTables(ctx context.Context, ids []string) error {
	if err := s.repo.deleteManyTables(ctx, ids); err != nil {
		return err
	}
	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      "",
		Action:      "delete_many",
		EntityType:  "table",
		EntityID:    "",
		Description: fmt.Sprintf("soft-deleted %d tables", len(ids)),
		Metadata: map[string]any{
			"ids": ids,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: "",
	})
	return nil
}

func (s *service) ListTables(ctx context.Context, q Table) ([]Table, error) {
	return s.repo.listTables(ctx, q)
}

func (s *service) DeleteManyColumns(ctx context.Context, ids []string) error {
	if err := s.repo.deleteManyColumns(ctx, ids); err != nil {
		return err
	}
	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      "",
		Action:      "delete_many",
		EntityType:  "column",
		EntityID:    "",
		Description: fmt.Sprintf("soft-deleted %d columns", len(ids)),
		Metadata: map[string]any{
			"ids": ids,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: "",
	})
	return nil
}

func (s *service) DeleteManySchemas(ctx context.Context, ids []string) error {
	if err := s.repo.deleteManySchemas(ctx, ids); err != nil {
		return err
	}
	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      "",
		Action:      "delete_many",
		EntityType:  "schema",
		EntityID:    "",
		Description: fmt.Sprintf("soft-deleted %d schemas", len(ids)),
		Metadata: map[string]any{
			"ids": ids,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: "",
	})
	return nil
}

func (s *service) List(ctx context.Context, filter Database, opts utils.ListOpts) ([]Database, error) {
	return s.repo.list(ctx, filter, opts)
}

func (s *service) ListSchemas(ctx context.Context, q Schema) ([]Schema, error) {
	return s.repo.listSchemas(ctx, q)
}

func (s *service) ListColumns(ctx context.Context, q Column) ([]Column, error) {
	return s.repo.listColumns(ctx, q)
}

func (s *service) GetSchema(ctx context.Context, q Schema) (*Schema, error) {
	found, exist, err := s.repo.getSchema(ctx, q)
	if err != nil {
		return nil, err
	}
	if !exist || found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (s *service) Get(ctx context.Context, application *Database) (*Database, error) {
	found, exist, err := s.repo.get(ctx, *application)
	if err != nil {
		return nil, err
	}
	if !exist || found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (s *service) GetTable(ctx context.Context, table *Table) (*Table, error) {
	found, exist, err := s.repo.getTable(ctx, *table)
	if err != nil {
		return nil, err
	}
	if !exist || found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (s *service) update(ctx context.Context, userID, databaseID string, p UpdateDatabasePayload) error {
	d, ok, err := s.repo.get(ctx, Database{ID: databaseID})
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: d.ApplicationID})
	if err != nil {
		return err
	}

	if app.UserID != userID {
		return ErrUnauthorized
	}

	if p.Database != "" {
		host := p.Host
		if host == "" {
			host = d.Host
		}
		_, dup, err := s.repo.get(ctx, Database{
			Database:      p.Database,
			Host:          host,
			ApplicationID: d.ApplicationID,
		})
		if err != nil {
			return err
		}
		if dup && (p.Database != d.Database || host != d.Host) {
			return ErrDuplicateDBName
		}
	}

	host := coalesceStr(p.Host, d.Host)
	port := coalesceUint(p.Port, d.Port)
	dbName := coalesceStr(p.Database, d.Database)
	user := coalesceStr(p.Username, d.Username)
	engine := coalesceStr(p.DbEngine, d.DbEngine)
	name := coalesceStr(p.Name, d.Name)

	var currentPwd string
	rawPwd, err := utils.Decrypt(d.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return err
	}
	currentPwd = string(rawPwd)
	pass := coalesceStr(p.Password, currentPwd)

	dbConn, err := utils.TestDatabaseConnection(utils.DatabaseConnectionPayload{
		Host:     host,
		Port:     port,
		Database: dbName,
		Username: user,
		Password: pass,
		DbEngine: engine,
	})
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}
	_ = dbConn.Close()

	encPwd, err := utils.Encrypt([]byte(pass), []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"Host":     host,
		"Port":     port,
		"Database": dbName,
		"Username": user,
		"Password": encPwd,
		"DbEngine": engine,
		"Name":     name,
	}
	if err := s.repo.update(ctx, d, updates); err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "update",
		EntityType:  "database",
		EntityID:    d.ID,
		Description: "updated database",
		Metadata: map[string]any{
			"updates": updates,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: d.ApplicationID,
	})

	credsChanged := host != d.Host ||
		port != d.Port ||
		dbName != d.Database ||
		user != d.Username ||
		pass != currentPwd ||
		engine != d.DbEngine

	if credsChanged {
		task := utils.DatabaseTask{
			DatabaseID: d.ID,
			UserID:     userID,
			Action:     utils.FetchDBAction,
		}
		payload, err := json.Marshal(task)
		if err == nil {
			_ = utils.PublishMessageToQueue(ctx, s.qConn, payload, utils.DATABASE_QUEUE)
		}
	}

	return nil
}

func (s *service) delete(ctx context.Context, userID, databaseID string) error {
	d, ok, err := s.repo.get(ctx, Database{ID: databaseID})
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: d.ApplicationID, UserID: userID})
	if err != nil {
		return err
	}
	if app.UserID != userID {
		return ErrUnauthorized
	}

	task := utils.DatabaseTask{
		DatabaseID: d.ID,
		UserID:     userID,
		Action:     utils.DeleteDBResourceAction,
	}
	payload, err := json.Marshal(task)
	if err == nil {
		_ = utils.PublishMessageToQueue(ctx, s.qConn, payload, utils.DATABASE_QUEUE)
	}

	if err := s.repo.delete(ctx, &Database{ID: d.ID}); err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "delete",
		EntityType:  "database",
		EntityID:    d.ID,
		Description: "soft-deleted database",
		Metadata: map[string]any{
			"database_id": d.ID,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: d.ApplicationID,
	})

	return nil
}

func (s *service) add(ctx context.Context, userID, appID string, payload AddDatabasePayload) error {
	_, err := s.applicationSvc.Get(ctx, &application.Application{ID: appID, UserID: userID})
	if err != nil {
		return err
	}

	if _, dup, err := s.repo.get(ctx, Database{
		Database:      payload.Database,
		Host:          payload.Host,
		ApplicationID: appID,
		Name:          payload.Name,
	}); err != nil {
		return err
	} else if dup {
		return ErrDuplicateDBName
	}

	conn, err := utils.TestDatabaseConnection(utils.DatabaseConnectionPayload{
		Host:     payload.Host,
		Port:     payload.Port,
		Database: payload.Database,
		Username: payload.Username,
		Password: payload.Password,
		DbEngine: payload.DbEngine,
	})
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}
	_ = conn.Close()

	encPwd, err := utils.Encrypt([]byte(payload.Password), []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return err
	}

	databaseData := &Database{
		Name:          payload.Name,
		Host:          payload.Host,
		Port:          payload.Port,
		Database:      payload.Database,
		Username:      payload.Username,
		Password:      encPwd,
		DbEngine:      payload.DbEngine,
		ApplicationID: appID,
		UserID:        userID,
	}
	created, err := s.repo.create(ctx, databaseData)
	if err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "create",
		EntityType:  "database",
		EntityID:    created.ID,
		Description: "created database",
		Metadata: map[string]any{
			"name":           created.Name,
			"host":           created.Host,
			"port":           created.Port,
			"database":       created.Database,
			"username":       created.Username,
			"db_engine":      created.DbEngine,
			"application_id": created.ApplicationID,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: created.ApplicationID,
	})

	task := utils.DatabaseTask{
		DatabaseID: created.ID,
		UserID:     userID,
		Action:     utils.FetchDBAction,
	}
	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return utils.PublishMessageToQueue(ctx, s.qConn, bytes, utils.DATABASE_QUEUE)
}

func coalesceStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func coalesceUint(a, b uint) uint {
	if a != 0 {
		return a
	}
	return b
}
