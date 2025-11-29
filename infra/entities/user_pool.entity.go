package entity

import "time"

type UserPool struct {
	ID   string `db:"id"`
	Name string `db:"name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type UserPoolWithApps struct {
	UserPool
	Apps []App `db:"apps"`
}
