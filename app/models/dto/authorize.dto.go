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

type RefreshResponse struct {
	SessionId        string    `json:"session_id"`
	Token            string    `json:"token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`

	User entity.User `json:"user"`
}
