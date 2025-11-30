package entity

import (
	"encoding/json"
	"time"
)

type App struct {
	ID          string `db:"id"`
	UsersPoolId string `db:"users_pool_id"`

	PublicKey string `db:"public_key"`
	SecretKey string `db:"secret_key"`

	Name string `db:"name"`

	LoginTypes          []string `db:"login_types"`
	TokenType           string   `db:"token_type"`
	TokenExpirationTime int64    `db:"token_expiration_time"`

	Private     bool `db:"private"`
	VerifyEmail bool `db:"verify_email"`

	Metadata json.RawMessage `db:"metadata"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type AppWithPool struct {
	App  `db:"app"`
	Pool UserPool `db:"users_pool"`
}
