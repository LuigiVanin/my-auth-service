package services

import (
	dto "auth_service/app/modules/core/profile/models"
	entity "auth_service/infra/entities"
)

// The permission algebra is not here: it is pure computation over two documents
// and lives in shared/permissions, so a guard or an entity can reach it without a
// dependency. See docs/steering/modules/profiles.md.
type IProfileService interface {
	// FindByIdVisibleTo replaces the unrestricted FindById. Returns nil, nil when the
	// profile does not exist OR is scoped to another organization - the caller cannot
	// tell the two apart, and must not.
	FindByIdVisibleTo(id string, organizationId string) (*entity.Profile, error)

	// FindByKey exists for one job: resolving the profile a new pool defaults to,
	// which is the only profile the code has to name without an id in hand. A key
	// is a seed handle, not an identifier callers pass around.
	FindByKey(key string) (*entity.Profile, error)

	FindAllVisibleTo(organizationId string, query *dto.GetProfilesQuery) (*dto.GetProfilesResponse, error)

	CreateForOrganization(caller *ProfileWriteContext, payload *dto.CreateProfile) (*entity.Profile, error)

	UpdateForOrganization(id string, caller *ProfileWriteContext, payload *dto.UpdateProfile) (*entity.Profile, error)
}

// ProfileWriteContext is the ceiling of whoever is writing. Both halves are
// required: checking only against the organization would let a member author a
// profile wider than the member itself holds.
type ProfileWriteContext struct {
	Organization *entity.Organization
	Participant  *entity.Participant
}
