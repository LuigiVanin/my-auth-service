package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type ProfileRepository struct {
	client *gorm.DB
}

var _ IProfileRepository = &ProfileRepository{}

func NewProfileRepository(client *gorm.DB) *ProfileRepository {
	return &ProfileRepository{
		client: client,
	}
}

func (r *ProfileRepository) FindProfileByAppRole(role string) ([]entity.AppRoleProfile, error) {
	var result []entity.AppRoleProfile

	err := r.client.Preload("Profile").Where("role = ?", role).Find(&result).Error

	return result, err
}
