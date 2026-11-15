package services

import (
	"time"

	dto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
)

type VerifyConsumableOtpResponse struct {
	Metadata          dto.OtpStoredMetadataPayload `json:"metadata"`
	VerificationCount int                          `json:"verification_count"`
	Verified          bool                         `json:"verified"`
}

// IssuedOtp carries the only copy of the secret in clear: the row stores it
// hashed outside development, so the caller delivers it now or never.
type IssuedOtp struct {
	Otp    *entity.Otp
	Secret string
}

type IOtpService interface {
	GenerateConsumable(
		action constants.AuthAction,
		app *entity.App,
		payload dto.ConsumableOtpPayload,
		ip string,
	) (*dto.GenerateConsumableOtpResponse, error)

	// Issue writes an OTP whose code is a 32 byte link secret instead of six digits,
	// valid for ttl, with no rate limit and no email - the caller delivers it.
	// ValidateConsumable and Invalidate consume it like any other.
	Issue(action constants.AuthAction, app *entity.App, contact string, ttl time.Duration) (*IssuedOtp, error)

	ValidateConsumable(payload dto.PayloadOtpData, appId string, action constants.AuthAction) (*entity.Otp, error)
	VerifyConsumable(otpId string, code string, appId string) (*VerifyConsumableOtpResponse, error)

	Invalidate(otpId string)
}
