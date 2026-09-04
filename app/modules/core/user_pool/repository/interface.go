package repository

import (
	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// UserPoolSearch describes the set of pools a listing is allowed to see.
//
// OrganizationId is not a pointer, for the same reason as AppSearch: a zero
// value must not compile into an empty WHERE over every pool in the database.
type UserPoolSearch struct {
	OrganizationId string

	// Optional case insensitive partial match.
	Name string
}

type IUserPoolRepository interface {
	FindOne(where entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error)
	Create(userPool entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error)
	Update(where entity.UsersPool, data dto.UserPoolUpdateDao, options ...repo.Option) (int64, error)

	// Dedicated queries: the ILIKE cannot be expressed by the typed `where`.
	FindSearch(search UserPoolSearch, options ...repo.Option) ([]entity.UsersPool, error)
	FindSearchCount(search UserPoolSearch, options ...repo.Option) (int64, error)
}
