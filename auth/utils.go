package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/showbaba/query-bridge/bridge/utils"
	"github.com/showbaba/query-bridge/shared"
	"golang.org/x/crypto/bcrypt"
)

func PasswordMatches(password, hash string) (bool, error) {
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

// GenerateToken generates a jwt token
func GenerateToken(JWTSecretKey, email string) (signedToken string, err error) {
	claims := &shared.AuthTokenJwtClaim{
		Email: email,
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

func ValidateGatewayToken() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("gateway_signature")
			if token == "" {
				shared.Dispatch400Error(w, "gateway-token not in header", nil)
				return
			}
			_, err := shared.ValidateGatewayToken(token, utils.GetConfig().AuthServiceSecretKey)
			if err != nil {
				shared.Dispatch400Error(w, "error validating gateway-token", fmt.Sprintf("%v", err))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
