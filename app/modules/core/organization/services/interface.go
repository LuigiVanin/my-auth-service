package services

import (
	dto "auth_service/app/modules/core/organization/models"
	entity "auth_service/infra/entities"
)

type IOrganizationService interface {
	// A pair because a user and its organization reference each other: the
	// organization is written first with no owner and stamped once the user exists.
	//
	// NOTE: nothing calls these today. A provisioning needs both writes on one
	// transaction and goes to the repository for that; these are here for the
	// routes that will perform the mutations on their own.
	Create(data dto.CreateOrganizationData) (*entity.Organization, error)
	SetOwner(organizationId string, ownerUserId uint) error

	// Returns nil, nil when it does not exist. Profile is preloaded.
	FindById(id string) (*entity.Organization, error)

	FindAllForUser(userId uint, query *dto.GetOrganizationsQuery) (*dto.GetOrganizationsResponse, error)

	Switch(user *entity.User, organizationId string) (*entity.Organization, error)

	CreateForUser(
		currentUser *entity.User,
		currentOrganization *entity.Organization,
		payload *dto.CreateOrganizationPayload,
	) (*entity.Organization, error)
}
