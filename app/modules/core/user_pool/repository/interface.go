package repository

import (
	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IUserPoolRepository interface {
	FindOne(where entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error)
	Create(userPool entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error)
	Update(where entity.UsersPool, data dto.UserPoolUpdateDao, options ...repo.Option) (int64, error)
}
