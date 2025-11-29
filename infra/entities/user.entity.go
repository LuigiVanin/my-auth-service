package entity

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string          `db:"id"`
	UsersPoolId  string          `db:"users_pool_id"`
	Name         string          `db:"name"`
	Email        string          `db:"email"`
	Phone        string          `db:"phone"`
	VerifyEmail  bool            `db:"verify_email"`
	PasswordHash string          `db:"password_hash"`
	Metadata     json.RawMessage `db:"metadata"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}
