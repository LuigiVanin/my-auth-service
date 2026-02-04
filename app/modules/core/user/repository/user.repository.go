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

	whereClause := r.client.Where(where)

	if len(with) > 0 {
		for _, relation := range with {
			whereClause = whereClause.Preload(relation)
		}
	}

	err := whereClause.First(&result).Error

	return &result, err
}

func (r *UserRepository) FindManyWhere(where entity.User, with ...string) (*[]entity.User, error) {
	var result []entity.User

	whereClause := r.client.Where(where)

	if len(with) > 0 {
		for _, relation := range with {
			whereClause = whereClause.Preload(relation)
		}
	}

	err := whereClause.Find(&result).Error

	return &result, err
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
