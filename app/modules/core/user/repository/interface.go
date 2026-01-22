package repository

import entity "auth_service/infra/entities"

type IUserRepository interface {
	FindWhere(where entity.User, with ...string) (*entity.User, error)
	FindManyWhere(where entity.User, with ...string) (*[]entity.User, error)
	Create(user entity.User) (*entity.User, error)
}
