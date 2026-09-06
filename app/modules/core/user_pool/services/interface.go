package services

import (
	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
)

type CreateUserPoolData struct {
	Name             string
	OwnerUserId      *uint
	OrganizationId   *string
	DefaultProfileId string
}

type IUserPoolService interface {
	// Refuses when DefaultProfileId grants more than caller holds in granter.
	Create(data CreateUserPoolData, granter *entity.Organization, caller *entity.Participant) (*entity.UsersPool, error)

	// Unscoped, and only safe where the id did not come from the caller.
	FindById(id string) (*entity.UsersPool, error)

	// The scoped lookup, for whenever the id comes from a request.
	FindByIdInOrganization(id string, organizationId string) (*entity.UsersPool, error)

	List(
		currentOrganization *entity.Organization,
		query *dto.UserPoolListQuery,
	) (*dto.GetUserPoolsResponse, error)
}
