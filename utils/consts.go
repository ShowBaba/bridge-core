package utils

import (
	"time"

	"gorm.io/gorm"
)

const (
	DbTimeout                                 = time.Second * 3
	keyEmail                           string = "email"
	keyID                              string = "id"
	NOTIFICATION_QUEUE                        = "NOTIFICATIONS_QUEUE"
	DATABASE_QUEUE                            = "DATABASE_QUEUE"
	MAIL_USERNAME                             = "noreply@bridge.com"
	QUERY_BRIDGE_MONGO_DB_NAME                = "query-bridge"
	QUERY_BRIDGE_MONGO_LOGS_COLLECTION        = "logs"
)

var DB *gorm.DB
