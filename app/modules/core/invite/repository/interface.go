package repository

import (
	"time"

	dto "auth_service/app/modules/core/invite/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

// No Update: every column after the insert is owned by the service, and the two
// transitions below are the only writers.
type IInviteRepository interface {
	FindOne(where entity.Invite, options ...repo.Option) (*entity.Invite, error)
	Create(invite entity.Invite, options ...repo.Option) (*entity.Invite, error)

	// An empty status lists every invite of the organization, newest first.
	FindForOrganization(organizationId string, status dto.InviteStatus, options ...repo.Option) ([]entity.Invite, error)
	FindForOrganizationCount(organizationId string, status dto.InviteStatus, options ...repo.Option) (int64, error)

	FindPendingFor(organizationId string, email string, options ...repo.Option) ([]entity.Invite, error)

	// Both match only a pending row, so 0 rows affected is a lost race or a row
	// already accepted, revoked or expired.
	Accept(id string, at time.Time, options ...repo.Option) (int64, error)
	Revoke(id string, at time.Time, options ...repo.Option) (int64, error)
}
