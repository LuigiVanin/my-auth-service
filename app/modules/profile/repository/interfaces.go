package repository

import (
	entity "auth_service/infra/entities"
)

type IProfileRepository interface {
	FindProfileByAppRole(role string) ([]entity.AppRoleProfile, error)
}
