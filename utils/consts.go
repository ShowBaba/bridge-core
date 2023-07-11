package utils

import (
	"time"

	"gorm.io/gorm"
)

const (
	DbTimeout                         = time.Second * 3
	keyEmail                   string = "email"
	keyID                      string = "id"
	NOTIFICATION_QUEUE                = "NOTIFICATIONS"
	DATABASE_QUEUE                    = "DATABASE"
	MAIL_USERNAME                     = "noreply@bridge.com"
	QUERY_BRIDGE_MONGO_DB_NAME        = "query-bridge"
)

var DB *gorm.DB
