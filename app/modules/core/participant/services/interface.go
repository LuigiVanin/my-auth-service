package services

import (
	dto "auth_service/app/modules/core/participant/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
)

// ResolvedParticipation is a participation together with what its profile grants
// once the ceiling of the organization is applied.
type ResolvedParticipation struct {
	Participant *entity.Participant
	Permissions *permissions.Resolved
}

type IParticipantService interface {
	// NOTE: nothing calls this today. A write that belongs to a unit of work goes
	// to the repository, so it shares the transaction; this is here for the routes
	// that will add a participant on their own.
	Create(organizationId string, userId uint, profileId string) (*entity.Participant, error)

	// Returns nil, nil when the user does not participate. Profile is preloaded.
	FindForUserInOrganization(organizationId string, userId uint) (*entity.Participant, error)

	// The organization is not necessarily the current one, so the OrganizationGuard
	// cannot vouch for the caller: requestingUserId is checked first.
	FindAllInOrganization(
		organizationId string,
		requestingUserId uint,
		query *dto.GetParticipantsQuery,
	) (*dto.GetParticipantsResponse, error)

	// FindForCurrentOrganization is what a single-user response reports. Every such
	// answer - login, register, refresh - goes through it, so no flow can be left
	// without one and none of them repeats the resolving rule.
	FindForCurrentOrganization(user *entity.User) (*ResolvedParticipation, error)
}
