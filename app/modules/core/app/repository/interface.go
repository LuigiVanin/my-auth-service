package repository

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// AppSearch describes the set of apps a listing is allowed to see, so the SQL
// stays in the repository while the service still says what it wants.
//
// OrganizationId is not a pointer on purpose: an earlier version assembled the
// condition in the service and produced an empty WHERE for anyone who was neither
// ADMIN nor MANAGER, which listed every app in the database.
type AppSearch struct {
	OrganizationId string

	// Optional case insensitive partial match.
	Name string
}

type IAppRepository interface {
	FindOne(where entity.App, options ...repo.Option) (*entity.App, error)
	Create(app entity.App, options ...repo.Option) (*entity.App, error)
	Update(where entity.App, data dto.AppUpdateDao, options ...repo.Option) (int64, error)

	// Dedicated queries: the visible set mixes an OR with an ILIKE, which the
	// typed `where` cannot express.
	FindSearch(search AppSearch, options ...repo.Option) ([]entity.App, error)
	FindSearchCount(search AppSearch, options ...repo.Option) (int64, error)
}
