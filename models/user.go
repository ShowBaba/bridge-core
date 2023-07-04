package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/showbaba/query-bridge/bridge/utils"
)

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Firstname string    `json:"firstname,omitempty"`
	Lastname  string    `json:"lastname,omitempty"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) Insert(db *sql.DB) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), utils.DbTimeout)
	defer cancel()
	var id int
	query := `INSERT INTO Users (email, firstname, lastname, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	if err := db.QueryRowContext(ctx, query,
		&u.Email, &u.Firstname,
		&u.Lastname, &u.Password,
		time.Now(), time.Now()).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (u *User) Update(db *sql.DB) (*User, error) {
	return nil, nil
}

func (u *User) Delete(db *sql.DB) error {
	return nil
}

func (u *User) DeleteByID(db *sql.DB, id int) error {
	return nil
}

func (u *User) GetAll(db *sql.DB) ([]*User, error) {
	return nil, nil
}

func (u *User) GetByEmail(db *sql.DB, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), utils.DbTimeout)
	defer cancel()
	query := `SELECT id, email, firstname, lastname, password, created_at, updated_at FROM Users WHERE email = $1`
	var user User
	row := db.QueryRowContext(ctx, query, email)
	if err := row.Scan(&user.ID, &user.Email, &user.Firstname, &user.Lastname, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		return nil, nil
	}
	return &user, nil
}

func (u *User) GetByID(idb *sql.DB, d int) (*User, error) {
	return nil, nil
}
