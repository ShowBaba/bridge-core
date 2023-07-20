package db

import (
	"github.com/showbaba/query-bridge/bridge-core/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	db.AutoMigrate(
		&models.Application{},
		&models.User{},
		&models.Database{},
		&models.Schema{},
		&models.Table{},
		&models.Column{},
		&models.Endpoint{},
	)

	db.Model(&models.Application{}).Association("UserID")
	db.Model(&models.Database{}).Association("ApplicationID")
	db.Model(&models.Schema{}).Association("DatabaseID")
	db.Model(&models.Table{}).Association("SchemaID")
	db.Model(&models.Column{}).Association("TableID")
	db.Model(&models.Endpoint{}).Association("TableID")
}