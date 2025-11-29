package services

import (
	upr "auth_service/app/modules/user_pool/repository"
)

type RegisterService struct {
	userPoolRepository upr.IUserPoolRepository
}

func NewRegisterService(userPoolRepository upr.IUserPoolRepository) IRegisterService {
	return &RegisterService{
		userPoolRepository: userPoolRepository,
	}
}

func (service *RegisterService) Register() error {

	return nil
}
