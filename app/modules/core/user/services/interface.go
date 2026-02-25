package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type IUserService interface {
	IsAlreadyCreated(email string, app *entity.App) (bool, error)
	FindUserInPool(email string, usersPoolId string) (*entity.User, error)
	Update(user *entity.User) error
	FindAllUsersFromApp(targetAppId string, currentUser *entity.User, targetApp *entity.App, query *dto.GetUsersAppQuery) (*dto.GetUsersAppResponse, error)
}
