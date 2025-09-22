package utils

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
)

type AuthTokenJwtClaim struct {
	Email string
	ID    string
	jwt.StandardClaims
}

type APIResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Mail struct {
	Sender  string
	To      []string
	Subject string
	Body    string
}

func (mail *Mail) BuildMessage() string {
	msg := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg += fmt.Sprintf("From: %s\r\n", mail.Sender)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(mail.To, ";"))
	msg += fmt.Sprintf("Subject: %s\r\n", mail.Subject)
	msg += fmt.Sprintf("\r\n%s\r\n", mail.Body)
	return msg
}

type DatabaseTask struct {
	DatabaseID    string
	ApplicationID string
	UserID        string
	Action        DATABASE_TASK_ACTION
}

type DATABASE_TASK_ACTION string

const (
	DeleteApplicationResourceAction DATABASE_TASK_ACTION = "delete_application_resource"
	DeleteDBResourceAction          DATABASE_TASK_ACTION = "delete_db_resource"
	UpdateAction                    DATABASE_TASK_ACTION = "update_database"
	FetchDBAction                   DATABASE_TASK_ACTION = "fetch_database"
)

type DatabaseConnectionPayload struct {
	Name     string `json:"name" validate:"required"`
	Host     string `json:"host" validate:"required"`
	Port     uint   `json:"port" validate:"required"`
	Database string `json:"database" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	DbEngine string `json:"db_engine" validate:"required"`
	SSLMode  string `json:"ssl_mode" default:"disable"` // "disable", "require", "verify-ca", "verify-full"
}

type SchemaData struct {
	Schema string
	Tables []TableData
}

type ColumnMeta struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
	IsPrimaryKey bool
	FKRefSchema  *string
	FKRefTable   *string
	FKRefColumn  *string
	FKConstraint *string
}

type IndexMeta struct {
	Name       string
	Definition string
	IsUnique   bool
}

type TableData struct {
	Table   string
	Columns []ColumnMeta
	Indexes []IndexMeta
}

type ListOpts struct {
	Limit     int
	Offset    int
	OrderBy   string
	OrderDesc bool
}
