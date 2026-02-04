package services

import (
	entity "auth_service/infra/entities"
	"errors"
	"strings"

	ur "auth_service/app/modules/core/user/repository"

	"gorm.io/gorm"
)

type UserService struct {
	userRepository ur.IUserRepository
}

var _ IUserService = &UserService{}

func NewUserService(userRepository ur.IUserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

func (this *UserService) IsAlreadyCreated(email string, app *entity.App) (bool, error) {
	_, err := this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err == nil {
		return true, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return false, err
}

func (this *UserService) FindUserInPool(email string, usersPoolId string) (*entity.User, error) {
	return this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: usersPoolId,
	})
}

func (this *UserService) Update(user *entity.User) error {
	return this.userRepository.Update(user)
}
