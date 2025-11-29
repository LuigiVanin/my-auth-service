package entity

import (
	"encoding/json"
	"time"
)

type App struct {
	ID                  string          `db:"id"`
	Name                string          `db:"name"`
	UsersPoolId         string          `db:"users_pool_id"`
	Code                string          `db:"code"`
	SecretKey           string          `db:"secret_key"`
	Private             bool            `db:"private"`
	LoginTypes          []string        `db:"login_types"`
	TokenType           string          `db:"token_type"`
	TokenExpirationTime int64           `db:"token_expiration_time"`
	Metadata            json.RawMessage `db:"metadata"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
