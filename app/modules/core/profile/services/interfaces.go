package services

import entity "auth_service/infra/entities"

type IProfileService interface {
	GetProfileByAppRole(role string) (*entity.Profile, error)
}
