package dto

import (
	entity "auth_service/infra/entities"
	"time"
)

type LoginPayloadWithPassoword struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginPayloadWithOtp struct {
	Email string `json:"email" validate:"required,email"`

	Otp PayloadOtpData `json:"otp"`
}

// type OtpPayload struct {
// 	OtpCode string `json:"otp_code" validate:"required"`
// 	OtpId   string `json:"otp_id" validate:"required"`
// }

type RequestInfo struct {
	IpAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

type LoginResponse struct {
	SessionId        string    `json:"session_id"`
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`

	User entity.User `json:"user"`
}
