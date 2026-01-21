package services

import (
	"auth_service/common/constants"
	entity "auth_service/infra/entities"
)

type IOtpService interface {
	GenerateConsumable(action constants.AuthAction) (*entity.Otp, error)
}
