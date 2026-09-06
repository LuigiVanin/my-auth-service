package repository

import (
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IProfileRepository interface {
	FindOne(where entity.Profile, options ...repo.Option) (*entity.Profile, error)

	FindByKey(key string, options ...repo.Option) (*entity.Profile, error)
}
