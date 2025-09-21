// application/service.go
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	auditpkg "github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

var (
	ErrDuplicateName = errors.New("duplicate application name")
	ErrNotFound      = errors.New("application not found")
)

type Service interface {
	Get(ctx context.Context, application *Application) (*Application, error)
	create(ctx context.Context, userID string, payload CreateApplicationPayload) error
	update(ctx context.Context, userID, appID string, payload UpdateApplicationPayload) error
	delete(ctx context.Context, userID, appID string) error
}

type service struct {
	repo     Repository
	qConn    *amqp091.Connection
	auditSvc auditpkg.Service
}

func NewService(repo Repository, qConn *amqp091.Connection, auditSvc auditpkg.Service) Service {
	return &service{repo, qConn, auditSvc}
}

func (s *service) Get(ctx context.Context, application *Application) (*Application, error) {
	found, exist, err := s.repo.get(ctx, *application)
	if err != nil {
		return nil, err
	}
	if !exist || found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (s *service) create(ctx context.Context, userID string, payload CreateApplicationPayload) error {
	if _, exist, err := s.repo.get(ctx, Application{Name: payload.Name, UserID: userID}); err != nil {
		return err
	} else if exist {
		return ErrDuplicateName
	}

	var encAPIKey string
	if payload.ApiKey != "" {
		e, err := utils.Encrypt([]byte(payload.ApiKey), []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			return err
		}
		encAPIKey = e
	}

	app := &Application{
		Name:   payload.Name,
		UserID: userID,
		ApiKey: encAPIKey,
	}
	if err := s.repo.create(ctx, app); err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
		UserID:        userID,
		Action:        "application.create",
		EntityType:    "application",
		EntityID:      app.ID,
		Description:   fmt.Sprintf("Created application %q", app.Name),
		Metadata:      map[string]any{"name": app.Name},
		ApplicationID: app.ID,
	})

	return nil
}

func (s *service) update(ctx context.Context, userID, appID string, payload UpdateApplicationPayload) error {
	found, exist, err := s.repo.get(ctx, Application{ID: appID, UserID: userID})
	if err != nil {
		return err
	}
	if !exist || found == nil {
		return ErrNotFound
	}

	upd := Application{}
	meta := map[string]any{}

	if payload.Name != "" && payload.Name != found.Name {
		if _, dup, err := s.repo.get(ctx, Application{Name: payload.Name, UserID: userID}); err != nil {
			return err
		} else if dup {
			return ErrDuplicateName
		}
		meta["old_name"] = found.Name
		meta["new_name"] = payload.Name
		upd.Name = payload.Name
	}

	if payload.ApiKey != "" {
		enc, err := utils.Encrypt([]byte(payload.ApiKey), []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			return err
		}
		upd.ApiKey = enc
		meta["api_key_updated"] = true
	}

	if err := s.repo.update(ctx, found, upd); err != nil {
		return err
	}

	if len(meta) == 0 {
		meta["no_fields_changed"] = true
	}
	fmt.Println("creating audit log with meta:", meta)
	_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
		UserID:        userID,
		Action:        "application.update",
		EntityType:    "application",
		EntityID:      found.ID,
		Description:   "Updated application",
		Metadata:      meta,
		ApplicationID: found.ID,
	})

	return nil
}

func (s *service) delete(ctx context.Context, userID, appID string) error {
	found, exist, err := s.repo.get(ctx, Application{ID: appID, UserID: userID})
	if err != nil {
		return err
	}
	if !exist || found == nil {
		return ErrNotFound
	}

	task := utils.DatabaseTask{
		UserID:        userID,
		ApplicationID: found.ID,
		Action:        utils.DeleteApplicationResourceAction,
	}
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	if err := utils.PublishMessageToQueue(ctx, s.qConn, payload, utils.DATABASE_QUEUE); err != nil {
		return err
	}

	if err := s.repo.delete(ctx, &Application{ID: found.ID}); err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
		UserID:        userID,
		Action:        "application.delete",
		EntityType:    "application",
		EntityID:      found.ID,
		Description:   fmt.Sprintf("Deleted application %q", found.Name),
		Metadata:      map[string]any{"soft_deleted": true},
		ApplicationID: found.ID,
	})

	return nil
}
