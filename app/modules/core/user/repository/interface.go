package repository

import entity "auth_service/infra/entities"

type IUserRepository interface {
	FindWhere(where entity.User, with ...string) (*entity.User, error)
	FindManyWhere(where entity.User, skip int, limit int, with ...string) (*[]entity.User, int64, error)
	FindFromAppId(appId string, skip int, limit int, with ...string) ([]entity.User, int64, error)
	Create(user entity.User) (*entity.User, error)
	Update(user *entity.User) error
}
