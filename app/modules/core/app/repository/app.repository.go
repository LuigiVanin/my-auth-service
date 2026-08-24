package repository

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type AppRepository struct {
	repo.BaseRepository[entity.App, dto.AppUpdateDao]
}

var _ IAppRepository = &AppRepository{}

func NewAppRepository(client *gorm.DB) *AppRepository {
	return &AppRepository{
		BaseRepository: repo.NewBaseRepository[entity.App, dto.AppUpdateDao](client),
	}
}

// searchScope is the single definition of the listing predicate, shared by
// FindSearch and FindSearchCount so the two can never drift apart.
func searchScope(search AppSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		query = query.Where("apps.organization_id = ?", search.OrganizationId)

		if search.Name != "" {
			query = query.Where("apps.name ILIKE ?", "%"+search.Name+"%")
		}

		return query
	}
}

func (this *AppRepository) FindSearch(search AppSearch, options ...repo.Option) ([]entity.App, error) {
	result := []entity.App{}

	err := this.ListQuery(options...).Scopes(searchScope(search)).Find(&result).Error

	return result, err
}

func (this *AppRepository) FindSearchCount(search AppSearch, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).Scopes(searchScope(search)).Count(&count).Error

	return count, err
}
