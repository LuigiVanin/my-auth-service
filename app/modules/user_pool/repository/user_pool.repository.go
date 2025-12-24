package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type UserPoolRepository struct {
	client *gorm.DB
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *gorm.DB) IUserPoolRepository {
	return &UserPoolRepository{
		client: client,
	}
}

func (r *UserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UsersPool, error) {
	// Implementation was empty in original code.
	// Preserving empty implementation but complying with GORM structure.
	return nil, nil
}
