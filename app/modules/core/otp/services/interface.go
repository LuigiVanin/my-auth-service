package services

import (
	"auth_service/app/models/dto"
	"auth_service/common/constants"
	entity "auth_service/infra/entities"
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
