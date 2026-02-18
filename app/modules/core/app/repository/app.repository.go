package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type AppRepository struct {
	client *gorm.DB
}

var _ IAppRepository = &AppRepository{}

func NewAppRepository(client *gorm.DB) *AppRepository {
	return &AppRepository{
		client: client,
	}
}

func (this *AppRepository) FindAppbyIdWithPool(id string) (*entity.App, error) {
	var result entity.App

	err := this.client.Preload("UsersPool").Where("id = ?", id).First(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (this *AppRepository) FindWhere(where entity.App, with ...string) (*entity.App, error) {
	var result entity.App

	err := this.client.Where(where).First(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (this *AppRepository) Create(app *entity.App) (*entity.App, error) {
	err := this.client.Create(app).Error
	if err != nil {
		return nil, err
	}
	return app, nil
}
