package services

import (
	entity "auth_service/infra/entities"
)

// The permission algebra is not here: it is pure computation over two documents
// and lives in shared/permissions, so a guard or an entity can reach it without a
// dependency. See docs/steering/modules/profiles.md.
type IProfileService interface {
	// FindById is how a profile is referenced everywhere a caller names one.
	// Returns nil, nil when it does not exist.
	FindById(id string) (*entity.Profile, error)

	// FindByKey exists for one job: resolving the profile a new pool defaults to,
	// which is the only profile the code has to name without an id in hand. A key
	// is a seed handle, not an identifier callers pass around.
	FindByKey(key string) (*entity.Profile, error)
}
