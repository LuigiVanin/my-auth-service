package services

import (
	entity "auth_service/infra/entities"
)

type CreateUserPoolData struct {
	Name             string
	OwnerUserId      *uint
	OrganizationId   *string
	DefaultProfileId string
}

type IUserPoolService interface {
	// Validates DefaultProfileId against the ceiling of granter before writing.
	Create(data CreateUserPoolData, granter *entity.Organization) (*entity.UsersPool, error)

	// Unscoped, and only safe where the id did not come from the caller.
	FindById(id string) (*entity.UsersPool, error)

	// The scoped lookup, for whenever the id comes from a request.
	FindByIdInOrganization(id string, organizationId string) (*entity.UsersPool, error)
}
