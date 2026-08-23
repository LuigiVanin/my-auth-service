package repository

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// AppSearch describes the set of apps a listing is allowed to see.
//
// It exists so the SQL stays in the repository while the service still says
// what it wants. OwnerUserId is not a pointer on purpose: the previous version
// assembled the condition as a raw string in the service and produced an empty
// WHERE for users that were neither ADMIN nor MANAGER, which listed every app
// in the database.
type AppSearch struct {
	// OwnerUserId restricts the listing to apps owned by this user.
	OwnerUserId uint

	// OrChildrenOfAppId widens the listing to also include the apps under this
	// app, which is the ADMIN case: "apps I own, plus apps under my app".
	OrChildrenOfAppId *string

	// Name is an optional case insensitive partial match.
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
