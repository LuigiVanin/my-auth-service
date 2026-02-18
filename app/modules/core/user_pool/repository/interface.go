package repository

import entity "auth_service/infra/entities"

type IUserPoolRepository interface {
	FindByAppIdAndPoolId(id string, appId string) (*entity.UsersPool, error)
	Create(userPool *entity.UsersPool) (*entity.UsersPool, error)
	FindById(id string) (*entity.UsersPool, error)
}
