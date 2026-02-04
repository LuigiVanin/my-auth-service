package dto

import (
	entity "auth_service/infra/entities"
	"time"
)

type AuthorizeResponse struct {
	User entity.User `json:"user"`

	SessionId string `json:"user_id"`
	Appid     string `json:"app_id"`

	ExpiresAt time.Time `json:"expires_at"`
	TokenType string    `json:"token_type"`

	Authorized bool `json:"authorized"`
}

type RefreshResponse struct {
	SessionId        string    `json:"session_id"`
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`

	User entity.User `json:"user"`
}

type ResetPasswordPayload struct {
	Email       string         `json:"email" validate:"required,email"`
	NewPassword string         `json:"new_password" validate:"required"`
	Otp         PayloadOtpData `json:"otp" validate:"required"`
}
