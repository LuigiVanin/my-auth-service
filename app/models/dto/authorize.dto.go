package dto

import (
	entity "auth_service/infra/entities"
	"time"
)

type AuthorizeReponse struct {
	User entity.User `json:"user"`

	SessionId string `json:"user_id"`
	Appid     string `json:"app_id"`

	ExpiresAt time.Time `json:"expires_at"`
	TokenType string    `json:"token_type"`

	Authorized bool `json:"authorized"`
}
