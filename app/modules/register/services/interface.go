package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type IRegisterService interface {
	Register() error

	RegisterWithPassword(
		app *entity.App,
		userData dto.RegisterPayloadWithPassoword,
	) (*entity.User, error)

	RegisterWithOtp() error
}
