package services

import (
	"auth_service/app/models/dto"
	"auth_service/common/constants"
	entity "auth_service/infra/entities"
)

type IOtpService interface {
	GenerateConsumable(
		app *entity.App,
		payload dto.ConsumableOtpPayload,
	) (*dto.GenerateConsumableOtpResponse, error)

	ValidateConsumable(payload dto.PayloadOtpData, appId string, action constants.AuthAction) (*entity.Otp, error)

	Invalidate(otpId string)
}
