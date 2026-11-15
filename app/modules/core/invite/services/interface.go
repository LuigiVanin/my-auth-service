package services

import (
	dto "auth_service/app/modules/core/invite/models"
	entity "auth_service/infra/entities"
)

type IInviteService interface {
	// Create, List and Revoke act on the organization the caller is currently in,
	// and refuse any other id. The caller is its participation: it caps the profile
	// an invite may hand out.
	Create(
		organizationId string,
		currentOrganization *entity.Organization,
		caller *entity.Participant,
		inviter *entity.User,
		app *entity.App,
		payload *dto.CreateInvite,
	) (*dto.InviteResponse, error)

	List(
		organizationId string,
		currentOrganization *entity.Organization,
		query *dto.GetInvitesQuery,
	) (*dto.GetInvitesResponse, error)

	Revoke(organizationId string, inviteId string, currentOrganization *entity.Organization) (*dto.InviteResponse, error)

	// Resolve returns the pending invite the token proves, with Organization and
	// Profile preloaded. Unknown, malformed, wrong secret and another app's invite
	// all answer 404; accepted, revoked and expired answer 400.
	Resolve(token string, app *entity.App) (*entity.Invite, error)

	Describe(token string, app *entity.App) (*dto.InviteDescription, error)

	// Accept is the token alone, for an email that already has an account in the
	// pool: no signup, no session, only the participation.
	Accept(token string, app *entity.App) (*dto.AcceptInviteResponse, error)

	// AcceptFor consumes a resolved invite on behalf of a user of its pool - the
	// common end of Accept and of the signup by invite.
	AcceptFor(invite *entity.Invite, user *entity.User) (*dto.AcceptInviteResponse, error)
}
