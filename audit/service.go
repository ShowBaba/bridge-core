package audit

import (
	"context"
	"encoding/json"
)

type Service interface {
	Create(ctx context.Context, in LogInput) (*Audit, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo}
}

func (s *service) Create(ctx context.Context, in LogInput) (*Audit, error) {
	var meta string
	if in.Metadata != nil {
		b, _ := json.Marshal(in.Metadata)
		meta = string(b)
	}
	a := &Audit{
		UserID:        in.UserID,
		Action:        in.Action,
		EntityType:    in.EntityType,
		EntityID:      in.EntityID,
		Description:   in.Description,
		Metadata:      meta,
		IPAddress:     in.IPAddress,
		ApplicationID: in.ApplicationID,
	}
	return s.repo.create(ctx, a)
}
