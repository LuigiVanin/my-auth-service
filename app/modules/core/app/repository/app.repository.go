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

func (this *AppRepository) Create(app *entity.App) (*entity.App, error) {
	err := this.client.Create(app).Error
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (this *AppRepository) FindWhere(where entity.App, with ...string) (*entity.App, error) {
	var result entity.App

	err := this.client.Where(where).First(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (this *AppRepository) FindManyWhere(where any, args []any, with ...string) (*[]entity.App, error) {
	var result []entity.App

	query := this.client.Where(where, args...)

	if len(with) > 0 {
		for _, relation := range with {
			query = query.Preload(relation)
		}
	}

	err := query.Find(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (this *AppRepository) FindManyWhereAndCount(
	where any,
	args []any,
	skip int,
	limit int,
	with ...string,
) ([]entity.App, int64, error) {

	var result []entity.App
	var count int64

	query := this.client.Model(&entity.App{}).Where(where, args...)

	if len(with) > 0 {
		for _, relation := range with {
			query = query.Preload(relation)
		}
	}

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if skip >= 0 {
		query = query.Offset(skip)
	}

	err = query.Find(&result).Error

	if err != nil {
		return nil, 0, err
	}

	return result, count, nil
}

func (this *AppRepository) Update(id string, app entity.App) (*entity.App, error) {
	return nil, nil
}
