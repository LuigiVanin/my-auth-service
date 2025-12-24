package repository

import (
	entity "auth_service/infra/entities"
	"errors"

	"gorm.io/gorm"
)

type AppRepository struct {
	client *gorm.DB
}

var _ IAppRepository = &AppRepository{}

func NewAppRepository(client *gorm.DB) IAppRepository {
	return &AppRepository{
		client: client,
	}
}

func (r *AppRepository) FindAppbyIdWithPool(id string) (*entity.App, error) {
	var result entity.App

	err := r.client.Preload("UsersPool").Where("id = ?", id).First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("app not found")
		}
		return nil, err
	}

	return &result, nil
}
