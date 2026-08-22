package services

import (
	dto "auth_service/app/modules/login/models"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
)

type ILoginService interface {
	LoginWithPassword(
		app *entity.App,
		userData dto.LoginPayloadWithPassoword,
		request sharedDto.RequestInfo,
	) (*dto.LoginResponse, error)

	LoginWithOtp(
		app *entity.App,
		userData dto.LoginPayloadWithOtp,
		request sharedDto.RequestInfo,
	) (*dto.LoginResponse, error)
}
