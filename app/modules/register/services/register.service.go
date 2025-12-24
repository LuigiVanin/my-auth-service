package services

import (
	"auth_service/app/models/dto"
	ur "auth_service/app/modules/user/repository"
	upr "auth_service/app/modules/user_pool/repository"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"

	"go.uber.org/zap"
)

type RegisterService struct {
	userPoolRepository upr.IUserPoolRepository
	userRepository     ur.IUserRepository
	logger             *zap.Logger
}

func NewRegisterService(userPoolRepository upr.IUserPoolRepository, userRepository ur.IUserRepository, logger *zap.Logger) IRegisterService {
	return &RegisterService{
		userPoolRepository: userPoolRepository,
		userRepository:     userRepository,
		logger:             logger,
	}
}

func (this *RegisterService) RegisterWithPassword(appId string, usersPoolId string, userData dto.RegisterPayloadWithPassoword) (*entity.User, error) {

	this.logger.Info("RegisterWithPassword - Service should be implemented", zap.String("appId", appId), zap.Any("userData", userData))

	user, err := this.userRepository.FindWhere(entity.User{
		Email:       userData.Email,
		UsersPoolId: usersPoolId,
	})

	return user, err
}

func (this *RegisterService) RegisterWithOtp() error {

	return e.ThrowNotImplementedError("RegisterWithOtp is not implemented")
}

func (service *RegisterService) Register() error {

	return nil
}
