package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	auditpkg "github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidLogin = errors.New("invalid login")
)

type Service interface {
	login(ctx context.Context, payload LoginPayload) (string, error)
}

type service struct {
	userSvc  user.Service
	auditSvc auditpkg.Service
}

func NewService(userSvc user.Service, auditSvc auditpkg.Service) Service {
	return &service{userSvc, auditSvc}
}

func (s *service) login(ctx context.Context, payload LoginPayload) (string, error) {
	u, ok, err := s.userSvc.Get(ctx, user.User{Email: payload.Email})
	if err != nil {
		_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
			UserID:      "",
			Action:      "LOGIN_FAILED",
			EntityType:  "User",
			EntityID:    "",
			Description: fmt.Sprintf("Login attempt failed due to DB error for email %s", payload.Email),
			Metadata: map[string]interface{}{
				"email": payload.Email,
			},
			IPAddress: utils.GetIPAddressFromCtx(ctx),
		})
		return "", err
	}
	if !ok || u == nil {
		_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
			UserID:      "",
			Action:      "LOGIN_FAILED",
			EntityType:  "User",
			EntityID:    "",
			Description: fmt.Sprintf("Login attempt failed, user not found for email %s", payload.Email),
			Metadata: map[string]interface{}{
				"email": payload.Email,
			},
			IPAddress: utils.GetIPAddressFromCtx(ctx),
		})
		return "", ErrUserNotFound
	}

	match, err := passwordMatches(payload.Password, u.Password)
	if err != nil {
		return "", err
	}
	if !match {
		_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
			UserID:      u.ID,
			Action:      "LOGIN_FAILED",
			EntityType:  "User",
			EntityID:    u.ID,
			Description: "Invalid password provided",
			Metadata: map[string]interface{}{
				"email": payload.Email,
			},
			IPAddress: utils.GetIPAddressFromCtx(ctx),
		})
		return "", ErrInvalidLogin
	}

	token, err := generateJWTToken(utils.GetConfig().JWTSecretKey, payload.Email, u.ID)
	if err != nil {
		return "", err
	}

	_, _ = s.auditSvc.Create(ctx, auditpkg.LogInput{
		UserID:      u.ID,
		Action:      "LOGIN_SUCCESS",
		EntityType:  "User",
		EntityID:    u.ID,
		Description: "User logged in successfully",
		Metadata: map[string]interface{}{
			"email": payload.Email,
		},
		IPAddress: utils.GetIPAddressFromCtx(ctx),
	})

	return token, nil
}

func passwordMatches(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func generateJWTToken(JWTSecretKey, email string, id string) (signedToken string, err error) {
	claims := &utils.AuthTokenJwtClaim{
		Email: email,
		ID:    id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(24)).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err = token.SignedString([]byte(JWTSecretKey))
	if err != nil {
		return
	}
	return
}
