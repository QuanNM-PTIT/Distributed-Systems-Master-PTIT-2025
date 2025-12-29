package models

import "time"

type User struct {
	ID           int64     `db:"id"`
	UserID       string    `db:"user_id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}
