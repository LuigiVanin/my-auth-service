package services

import (
	dto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
)

type VerifyConsumableOtpResponse struct {
	Metadata          dto.OtpStoredMetadataPayload `json:"metadata"`
	VerificationCount int                          `json:"verification_count"`
	Verified          bool                         `json:"verified"`
}

type IOtpService interface {
	GenerateConsumable(
		action constants.AuthAction,
		app *entity.App,
		payload dto.ConsumableOtpPayload,
		ip string,
	) (*dto.GenerateConsumableOtpResponse, error)

	ValidateConsumable(payload dto.PayloadOtpData, appId string, action constants.AuthAction) (*entity.Otp, error)
	VerifyConsumable(otpId string, code string, appId string) (*VerifyConsumableOtpResponse, error)

	Invalidate(otpId string)
}
