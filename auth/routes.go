package auth

import (
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func InitializeAuthRoutes(router *mux.Router, dbClient *gorm.DB) {
	db = dbClient
	router.HandleFunc("/login", LoginHandler).Methods("POST")
}
