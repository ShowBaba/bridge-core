package db

import (
	"log"

	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/endpoint"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&application.Application{},
		&user.User{},
		&database.Database{},
		&database.Table{},
		&database.Schema{},
		&database.Column{},
		&database.Index{},
		&endpoint.Endpoint{},
		&audit.Audit{},
	); err != nil {
		log.Printf("auto-migrate failed: %v", err)
		return err
	}
	return nil
}
