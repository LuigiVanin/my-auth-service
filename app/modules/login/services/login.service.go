package services

import (
	"auth_service/app/models/dto"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"
)

type LoginService struct{}

func NewLoginService() ILoginService {
	return &LoginService{}
}

func (this *LoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword) (*entity.User, error) {

	return nil, e.ThrowNotImplementedError("LoginWithPassword is not implemented")
}

func (this *LoginService) LoginWithOtp(app *entity.App, userData dto.LoginPayloadWithOtp) (*entity.User, error) {

	return nil, e.ThrowNotImplementedError("LoginWithOtp is not implemented")
}
