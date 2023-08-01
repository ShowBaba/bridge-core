package utils

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rabbitmq/amqp091-go"
	"golang.org/x/crypto/bcrypt"
)

func ValidateAuthHeaderToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			Dispatch400Error(w, "auth token not in header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			Dispatch400Error(w, "bearer token not in header")
			return
		}
		claim, err := ValidateAuthToken(parts[1], GetConfig().JWTSecretKey)
		if err != nil {
			Dispatch400Error(w, fmt.Sprintf("error validating auth token token: %v", err))
			return
		}
		// set values in the request context
		ctx := context.WithValue(r.Context(), keyEmail, claim.Email)
		ctx = context.WithValue(r.Context(), keyID, claim.ID)
		r = r.WithContext(ctx)
		next(w, r)
	}
}

// validate auth token in header
func ValidateAuthToken(signedToken, SECRET_KEY string) (*AuthTokenJwtClaim, error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&AuthTokenJwtClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AuthTokenJwtClaim)
	if !ok {
		return nil, err
	}
	// check the expiration date of the token
	if claims.ExpiresAt < time.Now().Local().Unix() {
		return nil, err
	}
	return claims, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func WriteError(statusCode int, message string) []byte {
	response := APIResponse{
		Status:  statusCode,
		Message: message,
	}
	data, err := json.Marshal(response)
	if err == nil {
		return data
	} else {
		log.Printf("Err: %s", err)
	}
	return nil
}

func WriteInfo(format string, args ...interface{}) []byte {
	response := map[string]string{
		"info": fmt.Sprintf(format, args...),
	}
	if data, err := json.Marshal(response); err == nil {
		return data
	} else {
		log.Printf("Err: %s", err)
	}
	return nil
}

func TestDatabaseConnection(payload DatabaseConnectionPayload) (*sql.DB, error) {
	var dsn string
	switch payload.DbEngine {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", payload.Username, payload.Password, payload.Host, payload.Port, payload.Database)
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", payload.Host, payload.Port, payload.Username, payload.Password, payload.Database)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", payload.DbEngine)
	}

	db, err := sql.Open(payload.DbEngine, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping the database: %v", err)
	}

	return db, nil
}

func PublishMessageToQueue(ctx context.Context, conn *amqp091.Connection, message []byte, queueName string) error {
	fmt.Println("publishing to ", queueName)
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(q.Name, q.Name, queueName, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = ch.PublishWithContext(
		ctx,
		"",
		q.Name,
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(ctx, queueName, "", false, false, amqp091.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	},
	)

	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	paddedPlaintext := padPlaintext(plaintext, aes.BlockSize)

	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	encryptedData := append(iv, ciphertext...)

	encodedData := base64.StdEncoding.EncodeToString(encryptedData)

	return encodedData, nil
}

func Decrypt(ciphertext string, key []byte) ([]byte, error) {
	encryptedData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	if len(encryptedData) < aes.BlockSize {
		return nil, errors.New("invalid ciphertext")
	}

	iv := encryptedData[:aes.BlockSize]
	encryptedText := encryptedData[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	decryptedText := make([]byte, len(encryptedText))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decryptedText, encryptedText)

	plaintext, err := unpadPlaintext(decryptedText)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func padPlaintext(plaintext []byte, blockSize int) []byte {
	padding := blockSize - (len(plaintext) % blockSize)
	paddedPlaintext := append(plaintext, bytes.Repeat([]byte{byte(padding)}, padding)...)
	return paddedPlaintext
}

func unpadPlaintext(paddedPlaintext []byte) ([]byte, error) {
	length := len(paddedPlaintext)
	if length == 0 {
		return nil, errors.New("invalid padded plaintext")
	}

	padding := int(paddedPlaintext[length-1])
	if padding > length {
		return nil, errors.New("invalid padding")
	}

	return paddedPlaintext[:length-padding], nil
}

func BoolPointer(b bool) *bool {
	return &b
}
