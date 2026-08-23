package repository

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IUserRepository interface {
	Find(where entity.User, options ...repo.Option) ([]entity.User, error)
	FindCount(where entity.User, options ...repo.Option) (int64, error)
	FindOne(where entity.User, options ...repo.Option) (*entity.User, error)
	Create(user entity.User, options ...repo.Option) (*entity.User, error)
	Update(where entity.User, data dto.UserUpdateDao, options ...repo.Option) (int64, error)
	Delete(where entity.User, options ...repo.Option) (int64, error)

	// Dedicated queries: the Find / FindCount pair is kept, and so is the option.
	FindFromAppId(appId string, options ...repo.Option) ([]entity.User, error)
	FindFromAppIdCount(appId string, options ...repo.Option) (int64, error)
}
