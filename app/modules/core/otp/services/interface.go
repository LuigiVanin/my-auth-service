package services

import (
	"auth_service/common/constants"
	entity "auth_service/infra/entities"
)

type IOtpService interface {
	GenerateConsumable(
		app *entity.App,
		action constants.AuthAction,
		metadata map[string]any,
	) (*entity.Otp, error)
}
