package repository

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type UserRepository struct {
	repo.BaseRepository[entity.User, dto.UserUpdateDao]
}

var _ IUserRepository = &UserRepository{}

func NewUserRepository(client *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: repo.NewBaseRepository[entity.User, dto.UserUpdateDao](client),
	}
}

// NOTE: a user belongs to an app through the pool the app points to, so this
// cannot be expressed by the typed `where` argument.
const userFromAppJoin = "JOIN apps ON apps.users_pool_id = users.users_pool_id"

func (this *UserRepository) FindFromAppId(appId string, options ...repo.Option) ([]entity.User, error) {
	result := []entity.User{}

	err := this.ListQuery(options...).
		Joins(userFromAppJoin).
		Where("apps.id = ?", appId).
		Find(&result).Error

	return result, err
}

func (this *UserRepository) FindFromAppIdCount(appId string, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).
		Joins(userFromAppJoin).
		Where("apps.id = ?", appId).
		Count(&count).Error

	return count, err
}
