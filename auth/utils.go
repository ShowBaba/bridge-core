package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/showbaba/query-bridge/bridge/utils"

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
func GenerateToken(JWTSecretKey, email string, id uint) (signedToken string, err error) {
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
