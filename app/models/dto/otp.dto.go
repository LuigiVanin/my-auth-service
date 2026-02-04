package dto

import (
	"auth_service/common/constants"
	"time"
)

type PayloadOtpData struct {
	Id   string `json:"id" validate:"required"`
	Code string `json:"code" validate:"required"`
}

// ConsumableOtpPayload is the internal structure used by the service
type ConsumableOtpPayload struct {
	Action constants.AuthAction `json:"action"`

	Contact string `json:"contact"`

	// Payload contains the actual data from the request body
	Payload map[string]any `json:"payload"`
}

type OtpStoredMetadataPayload struct {
	Email string  `json:"email" validate:"email"`
	Phone *string `json:"phone,omitempty"`
	Name  string  `json:"name,omitempty"`
}

type OtpStoredMetadata struct {
	Ip                string                   `json:"ip"`
	Payload           OtpStoredMetadataPayload `json:"payload"`
	VerificationCount int                      `json:"verification_count"`
}

// OtpRegisterPayload is the request body for REGISTER action
type OtpRegisterPayload struct {
	Email string  `json:"email" validate:"required,email"`
	Phone *string `json:"phone"`
	Name  string  `json:"name" validate:"required"`
}

// OtpLoginPayload is the request body for LOGIN action
type OtpLoginPayload struct {
	Email string `json:"email" validate:"required,email"`
}

// Legacy types kept for backward compatibility
type OtpMetadata struct {
	Ip              string `json:"ip"`
	VeficationCount int    `json:"vefication_count"`
}

type OtpMetadataPayload struct {
	Email string  `json:"email" validate:"email"`
	Phone *string `json:"phone"`
	Name  string  `json:"name"`
}

type OtpRegisterActionMetadata struct {
	OtpMetadata

	Payload OtpMetadataPayload `json:"payload" validate:"required"`
}

type OtpLoginActionMetadata struct {
	OtpMetadata

	Payload OtpMetadataPayload `json:"payload" validate:"required"`
}

type GenerateConsumableOtpResponse struct {
	OtpId string `json:"otp_id"`
	AppId string `json:"app_id"`

	Action constants.AuthAction `json:"action"`

	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	Payload map[string]any `json:"-"`
}

type VerifyConsumableOtpPayload struct {
	Code string `json:"code" validate:"required"`
}

type VerifyConsumableOtpResponse struct {
	Metadata          OtpStoredMetadataPayload `json:"metadata"`
	VerificationCount int                      `json:"verification_count"`
	Verified          bool                     `json:"verified"`
}
