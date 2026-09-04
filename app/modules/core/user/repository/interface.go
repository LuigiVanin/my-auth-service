package repository

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// UserSearch describes the set of users a listing is allowed to see.
//
// UsersPoolId is mandatory on purpose: every listing is scoped to one pool,
// which the service resolves and authorizes first, so an empty WHERE over the
// whole users table cannot be expressed.
type UserSearch struct {
	UsersPoolId string

	// Name and Email are optional case insensitive partial matches.
	Name  string
	Email string
}

type IUserRepository interface {
	Find(where entity.User, options ...repo.Option) ([]entity.User, error)
	FindCount(where entity.User, options ...repo.Option) (int64, error)
	FindOne(where entity.User, options ...repo.Option) (*entity.User, error)
	Create(user entity.User, options ...repo.Option) (*entity.User, error)
	Update(where entity.User, data dto.UserUpdateDao, options ...repo.Option) (int64, error)
	Delete(where entity.User, options ...repo.Option) (int64, error)

	// Dedicated queries: the name and email filters are ILIKE matches the typed
	// `where` cannot express.
	FindSearch(search UserSearch, options ...repo.Option) ([]entity.User, error)
	FindSearchCount(search UserSearch, options ...repo.Option) (int64, error)
}
