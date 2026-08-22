package services

import (
	dto "auth_service/app/modules/register/models"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
)

type IRegisterService interface {
	Register() error

	RegisterWithPassword(
		app *entity.App,
		userData dto.RegisterPayloadWithPassoword,
		request sharedDto.RequestInfo,
	) (*dto.RegisterResponse, error)

	RegisterWithOtp(
		app *entity.App,
		payload dto.RegisterPayloadWithOtp,
		request sharedDto.RequestInfo,
	) (*dto.RegisterResponse, error)
}
