package services

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
)

type IUserService interface {
	IsAlreadyCreated(email string, app *entity.App) (bool, error)
	FindUserInPool(email string, usersPoolId string) (*entity.User, error)
	Update(where entity.User, data dto.UserUpdateDao) (int64, error)

	List(
		currentUser *entity.User,
		currentOrganization *entity.Organization,
		query *dto.UserListQuery,
	) (*dto.GetUsersResponse, error)

	FindById(
		currentUser *entity.User,
		currentOrganization *entity.Organization,
		targetUserId uint,
	) (*entity.User, error)
}
