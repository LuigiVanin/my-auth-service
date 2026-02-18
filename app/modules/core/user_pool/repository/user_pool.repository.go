package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type UserPoolRepository struct {
	client *gorm.DB
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *gorm.DB) *UserPoolRepository {
	return &UserPoolRepository{
		client: client,
	}
}

func (r *UserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UsersPool, error) {
	// Implementation was empty in original code.
	// Preserving empty implementation but complying with GORM structure.
	return nil, nil
}

func (this *UserPoolRepository) Create(userPool *entity.UsersPool) (*entity.UsersPool, error) {
	err := this.client.Create(userPool).Error
	if err != nil {
		return nil, err
	}
	return userPool, nil
}

func (this *UserPoolRepository) FindById(id string) (*entity.UsersPool, error) {
	var result entity.UsersPool
	err := this.client.Where("id = ?", id).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}
