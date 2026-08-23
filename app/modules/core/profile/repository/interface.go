package repository

import (
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IProfileRepository interface {
	// Dedicated query: a role maps to a handful of rows in a reference table,
	// always read in full and always ordered by priority.
	FindByAppRole(role string, options ...repo.Option) ([]entity.AppRoleProfile, error)
}
