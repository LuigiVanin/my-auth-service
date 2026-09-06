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

// searchScope is the single definition of the listing predicate, shared by
// FindSearch and FindSearchCount so the two cannot drift apart.
func searchScope(search UserSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		query = query.Where("users.users_pool_id = ?", search.UsersPoolId)

		if search.Name != "" {
			query = query.Where("users.name ILIKE ?", "%"+search.Name+"%")
		}

		if search.Email != "" {
			query = query.Where("users.email ILIKE ?", "%"+search.Email+"%")
		}

		return query
	}
}

func (this *UserRepository) FindSearch(search UserSearch, options ...repo.Option) ([]entity.User, error) {
	result := []entity.User{}

	err := this.ListQuery(options...).Scopes(searchScope(search)).Find(&result).Error

	return result, err
}

func (this *UserRepository) FindSearchCount(search UserSearch, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).Scopes(searchScope(search)).Count(&count).Error

	return count, err
}
