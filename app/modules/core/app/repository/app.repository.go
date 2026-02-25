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

func (this *AppRepository) FindManyWhere(query any, args []any, with ...string) (*[]entity.App, error) {
	var result []entity.App

	where := this.client.Where(query, args...)

	if len(with) > 0 {
		for _, relation := range with {
			where = where.Preload(relation)
		}
	}

	err := where.Find(&result).Error

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (this *AppRepository) FindManyWhereAndCount(
	query any,
	args []any,
	skip int,
	limit int,
	with ...string,
) (*[]entity.App, int64, error) {

	var result []entity.App
	var count int64

	where := this.client.Model(&entity.App{}).Where(query, args...)

	if len(with) > 0 {
		for _, relation := range with {
			where = where.Preload(relation)
		}
	}

	err := where.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		where = where.Limit(limit)
	}
	if skip >= 0 {
		where = where.Offset(skip)
	}

	err = where.Find(&result).Error

	if err != nil {
		return nil, 0, err
	}

	return &result, count, nil
}
