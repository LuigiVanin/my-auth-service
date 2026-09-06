package repository

import (
	dto "auth_service/app/modules/core/profile/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// FindOne stays unscoped for the seed, which has no organization to scope by.
type IProfileRepository interface {
	FindOne(where entity.Profile, options ...repo.Option) (*entity.Profile, error)

	FindByKey(key string, options ...repo.Option) (*entity.Profile, error)

	FindVisibleTo(search ProfileSearch, options ...repo.Option) ([]entity.Profile, error)

	FindVisibleToCount(search ProfileSearch, options ...repo.Option) (int64, error)

	FindByIdVisibleTo(id string, organizationId string, options ...repo.Option) (*entity.Profile, error)

	Create(value entity.Profile, options ...repo.Option) (*entity.Profile, error)

	Update(where entity.Profile, dao dto.ProfileUpdateDao, options ...repo.Option) (int64, error)
}

// OrganizationId is who is looking, not what is being looked for. Scoped drops the
// global rows from the result.
type ProfileSearch struct {
	OrganizationId string
	Name           string
	Scoped         bool
}
