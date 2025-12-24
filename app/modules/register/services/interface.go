package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type IRegisterService interface {
	Register() error

	RegisterWithPassword(
		appId string,
		usersPoolId string,
		userData dto.RegisterPayloadWithPassoword,
	) (*entity.User, error)

	RegisterWithOtp() error
}
