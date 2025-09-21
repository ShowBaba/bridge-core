package db

import (
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/endpoint"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&application.Application{},
		&user.User{},
		&database.Database{},
		&database.Schema{},
		&database.Table{},
		&database.Column{},
		&endpoint.Endpoint{},
		&audit.Audit{},
	)

	db.Model(&application.Application{}).Association("UserID")
	db.Model(&database.Database{}).Association("ApplicationID")
	db.Model(&database.Schema{}).Association("DatabaseID")
	db.Model(&database.Table{}).Association("SchemaID")
	db.Model(&database.Column{}).Association("TableID")
	db.Model(&endpoint.Endpoint{}).Association("TableID")

	if err != nil {
		return err
	}
	return nil
}
