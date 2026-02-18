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
		request dto.RequestInfo,
	) (*dto.RegisterResponse, error)

	RegisterWithOtp(
		app *entity.App,
		payload dto.RegisterPayloadWithOtp,
		request dto.RequestInfo,
	) (*dto.RegisterResponse, error)
}
