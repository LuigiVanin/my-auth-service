package repository

import (
	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type UserPoolRepository struct {
	repo.BaseRepository[entity.UsersPool, dto.UserPoolUpdateDao]
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *gorm.DB) *UserPoolRepository {
	return &UserPoolRepository{
		BaseRepository: repo.NewBaseRepository[entity.UsersPool, dto.UserPoolUpdateDao](client),
	}
}

// searchScope is the single definition of the listing predicate, shared by
// FindSearch and FindSearchCount so the two cannot drift apart.
func searchScope(search UserPoolSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		query = query.Where("users_pool.organization_id = ?", search.OrganizationId)

		if search.Name != "" {
			query = query.Where("users_pool.name ILIKE ?", "%"+search.Name+"%")
		}

		return query
	}
}

func (this *UserPoolRepository) FindSearch(search UserPoolSearch, options ...repo.Option) ([]entity.UsersPool, error) {
	result := []entity.UsersPool{}

	err := this.ListQuery(options...).Scopes(searchScope(search)).Find(&result).Error

	return result, err
}

func (this *UserPoolRepository) FindSearchCount(search UserPoolSearch, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).Scopes(searchScope(search)).Count(&count).Error

	return count, err
}
