package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type ILoginService interface {
	LoginWithPassword(
		app *entity.App,
		userData dto.LoginPayloadWithPassoword,
	) (*entity.User, error)

	LoginWithOtp(
		app *entity.App,
		userData dto.LoginPayloadWithOtp,
	) (*entity.User, error)
}
