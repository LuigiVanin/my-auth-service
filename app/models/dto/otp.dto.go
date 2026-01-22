package dto

import (
	"auth_service/common/constants"
)

type ConsumableOtpPayload struct {
	Action constants.AuthAction `json:"action" validate:"required,oneof=LOGIN REGISTER VERIFY_EMAIL TWO_FA FORGOT_PASSWORD CHANGE_EMAIL REGEN_APP_SECRET_KEY"`

	// Metadata accepts any valid JSON object. Fiber's JSON parser will reject invalid JSON.
	// Use omitempty to make it optional, or remove it to require the field.
	Metadata map[string]any `json:"metadata" validate:"required"`
}
