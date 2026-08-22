package models

import (
	"encoding/json"
	"time"

	otpDto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
)

type RegisterPayloadWithPassoword struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	Name     string  `json:"name" validate:"required"`
	Phone    *string `json:"phone"`

	Metadata json.RawMessage `json:"metadata" validate:"required"`
}

type RegisterPayloadWithOtp struct {
	Email string  `json:"email" validate:"required,email"`
	Phone *string `json:"phone"`
	Name  string  `json:"name" validate:"required"`

	Metadata map[string]any `json:"metadata" validate:"required"`

	Password string `json:"password,omitempty" validate:"omitempty,min=8"`

	Otp otpDto.PayloadOtpData `json:"otp" validate:"required"`
}

type RegisterResponse struct {
	SessionId        string    `json:"session_id"`
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`

	User entity.User `json:"user"`
}
