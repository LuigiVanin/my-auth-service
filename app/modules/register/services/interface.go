package services

import (
	dto "auth_service/app/modules/register/models"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
	"auth_service/shared/permissions"
)

// ProvisionedUser is everything a provisioning writes. Permissions is what the
// participation resolves to against the ceiling of the organization, which is
// neither of the two profiles on their own.
type ProvisionedUser struct {
	User        *entity.User
	Participant *entity.Participant
	Permissions *permissions.Resolved
}

type IRegisterService interface {
	Register() error

	RegisterWithPassword(
		app *entity.App,
		userData dto.RegisterPayloadWithPassoword,
		request sharedDto.RequestInfo,
	) (*dto.RegisterResponse, error)

	RegisterWithOtp(
		app *entity.App,
		payload dto.RegisterPayloadWithOtp,
		request sharedDto.RequestInfo,
	) (*dto.RegisterResponse, error)

	// ProvisionUser writes a user together with the organization it owns and its
	// participation in it. Exposed because creating a user is not only something
	// registration does - an operator flow will need the same unit of work.
	ProvisionUser(app *entity.App, user entity.User) (*ProvisionedUser, error)
}
