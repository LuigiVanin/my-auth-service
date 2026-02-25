package repository

import (
	entity "auth_service/infra/entities"
	"errors"

	"gorm.io/gorm"
)

type UserRepository struct {
	client *gorm.DB
}

var _ IUserRepository = &UserRepository{}

func NewUserRepository(client *gorm.DB) *UserRepository {
	return &UserRepository{
		client: client,
	}
}

func (r *UserRepository) FindWhere(where entity.User, with ...string) (*entity.User, error) {
	var result entity.User

	query := r.client.Where(where)

	if len(with) > 0 {
		for _, relation := range with {
			query = query.Preload(relation)
		}
	}

	err := query.First(&result).Error

	return &result, err
}

func (r *UserRepository) FindManyWhere(where entity.User, skip int, limit int, with ...string) (*[]entity.User, int64, error) {
	var result []entity.User
	var count int64

	whereClause := r.client.Model(&entity.User{}).Where(where)

	if len(with) > 0 {
		for _, relation := range with {
			whereClause = whereClause.Preload(relation)
		}
	}

	err := whereClause.Count(&count).Error

	if err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		whereClause = whereClause.Limit(limit)
	}

	if skip >= 0 {
		whereClause = whereClause.Offset(skip)
	}

	err = whereClause.Find(&result).Error

	return &result, count, err
}

func (r *UserRepository) Create(user entity.User) (*entity.User, error) {
	err := r.client.Create(&user).Error
	return &user, err
}

func (r *UserRepository) Update(user *entity.User) error {

	if user.ID == 0 {
		return errors.New("user id is required")
	}

	return r.client.Save(user).Error
}

func (this *UserRepository) FindFromAppId(appId string, skip int, limit int, with ...string) ([]entity.User, int64, error) {

	var users []entity.User
	var count int64

	query := this.client.Table("users").Select("users.*")

	query = query.
		Joins("JOIN apps ON apps.users_pool_id = users.users_pool_id").
		Where("apps.id = ?", appId).
		Count(&count)

	if limit > 0 {

		query = query.Offset(skip)
	}

	if skip > 0 {
		query = query.Limit(limit)
	}

	if len(with) > 0 {
		for _, relation := range with {
			query = query.Preload(relation)
		}
	}

	err := query.Find(&users).Error
	/*
			SELECT u.*, a.name FROM users AS u
		    JOIN apps AS a ON u.users_pool_id = a.users_pool_id
		    	WHERE a.id = 'dc29ac9e-d39e-44c2-bc77-86ac2b059155'
		    	OFFSET 0
		    	LIMIT 10
	*/

	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}
