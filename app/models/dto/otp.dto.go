package dto

import (
	"auth_service/common/constants"
	"time"
)

type ConsumableOtpPayload struct {
	Action constants.AuthAction `json:"action" validate:"required,oneof=LOGIN REGISTER VERIFY_EMAIL TWO_FA FORGOT_PASSWORD CHANGE_EMAIL REGEN_APP_SECRET_KEY"`

	Contact string `json:"contact"`

	// Metadata accepts any valid JSON object. Fiber's JSON parser will reject invalid JSON.
	// Use omitempty to make it optional, or remove it to require the field.
	Metadata map[string]any `json:"metadata" validate:"required"`
}
type OtpMetadata struct {
	Ip              string `json:"ip"`
	VeficationCount int    `json:"vefication_count"`
}

type OtpRegisterActionMetadata struct {
	OtpMetadata

	Email string  `json:"email" validate:"required,email"`
	Phone *string `json:"phone"`
	Name  string  `json:"name" validate:"required"`
}

type OtpLoginActionMetadata struct {
	OtpMetadata

	Email string `json:"email" validate:"required,email"`
}

type OtpRegisterPayload struct {
	Action   constants.AuthAction      `json:"action" validate:"required,oneof=LOGIN REGISTER VERIFY_EMAIL TWO_FA FORGOT_PASSWORD CHANGE_EMAIL REGEN_APP_SECRET_KEY"`
	Metadata OtpRegisterActionMetadata `json:"metadata" validate:"required"`
}

type OtpLoginActionPayload struct {
	Action constants.AuthAction `json:"action" validate:"required,oneof=LOGIN REGISTER VERIFY_EMAIL TWO_FA FORGOT_PASSWORD CHANGE_EMAIL REGEN_APP_SECRET_KEY"`

	Metadata OtpLoginActionMetadata `json:"metadata" validate:"required"`
}

type GenerateConsumableOtpResponse struct {
	OtpId string `json:"otp_id"`
	AppId string `json:"app_id"`

	Action constants.AuthAction `json:"action"`

	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	Metadata any `json:"metadata"`
}

type PayloadOtpData struct {
	Id   string `json:"id" validate:"required"`
	Code string `json:"code" validate:"required"`
}
