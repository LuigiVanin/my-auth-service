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

	Otp OtpPayload `json:"otp"`
}

type OtpPayload struct {
	OtpCode string `json:"otpCode" validate:"required,min=6,max=6"`
	OtpId   string `json:"otpId" validate:"required"`
}

type RequestInfo struct {
	IpAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

type LoginResponse struct {
	SessionId        string    `json:"sessionId"`
	Token            string    `json:"token"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`

	User entity.User `json:"user"`
}
