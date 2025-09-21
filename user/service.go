package user

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rabbitmq/amqp091-go"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

var (
	ErrEmailInUse = errors.New("email already used")
	ErrNotFound   = errors.New("user not found")
)

type Service interface {
	register(ctx context.Context, p RegisterPayload) (*User, error)
	getProfile(ctx context.Context, userID string) (*User, error)
	Get(ctx context.Context, q User) (*User, bool, error)
}

type service struct {
	repo  Repository
	qConn *amqp091.Connection
}

func NewService(repo Repository, qConn *amqp091.Connection) Service {
	return &service{repo: repo, qConn: qConn}
}

func (s *service) Get(ctx context.Context, q User) (*User, bool, error) {
	return s.repo.get(ctx, q)
}

func (s *service) register(ctx context.Context, p RegisterPayload) (*User, error) {
	if _, ok, err := s.repo.get(ctx, User{Email: p.Email}); err != nil {
		return nil, err
	} else if ok {
		return nil, ErrEmailInUse
	}

	hash, err := utils.HashPassword(p.Password)
	if err != nil {
		return nil, err
	}

	u := &User{
		Email:     p.Email,
		FirstName: p.Firstname,
		LastName:  p.Lastname,
		Password:  hash,
	}
	created, err := s.repo.create(ctx, u)
	if err != nil {
		return nil, err
	}

	mail := utils.Mail{
		Sender:  utils.MAIL_USERNAME,
		Subject: "Welcome QueryBridge!",
		To:      []string{p.Email},
		Body: `<div style="font-family: Helvetica, Arial, sans-serif; min-width: 1000px; overflow: auto; line-height: 2;">
<div style="margin: 50px auto; width: 70%; padding: 20px 0;">
<div style="border-bottom: 1px solid #eee;"><a href="google.com" style="font-size: 1.4em; color: #00466a; text-decoration: none; font-weight: 600;">QueryBridge</a></div>
<p style="font-size: 1.1em;">Hi,</p>
<p>Hi ` + p.Firstname + `</p>
<p>Welcome to QueryBridge</p>
<p style="font-size: 0.9em;">Regards,<br />QueryBridge</p>
<hr style="border: none; border-top: 1px solid #eee;" />
</div>
</div>`,
	}
	payload, err := json.Marshal(mail)
	if err == nil {
		_ = utils.PublishMessageToQueue(ctx, s.qConn, payload, utils.NOTIFICATION_QUEUE)
	}

	return created, nil
}

func (s *service) getProfile(ctx context.Context, userID string) (*User, error) {
	u, ok, err := s.repo.get(ctx, User{ID: userID})
	if err != nil {
		return nil, err
	}
	if !ok || u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}
