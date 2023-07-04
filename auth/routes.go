package auth

import (
	"database/sql"

	"github.com/gorilla/mux"
)

var (
	db *sql.DB
)

func InitializeAuthRoutes(router *mux.Router, dbClient *sql.DB) {
	db = dbClient
	router.HandleFunc("/login", LoginHandler).Methods("POST")
}
