package utils

import (
	"database/sql"
	"time"
)

const DbTimeout = time.Second * 3

var DB *sql.DB
