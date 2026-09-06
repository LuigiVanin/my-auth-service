package repository

import (
	dto "auth_service/app/modules/core/profile/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type ProfileRepository struct {
	repo.BaseRepository[entity.Profile, dto.ProfileUpdateDao]
}

var _ IProfileRepository = &ProfileRepository{}

func NewProfileRepository(client *gorm.DB) *ProfileRepository {
	return &ProfileRepository{
		BaseRepository: repo.NewBaseRepository[entity.Profile, dto.ProfileUpdateDao](client),
	}
}

func (this *ProfileRepository) FindByKey(key string, options ...repo.Option) (*entity.Profile, error) {
	query := this.ListQuery(options...).Where("key = ?", key)

	return repo.FirstOrNil[entity.Profile](query)
}

func (this *ProfileRepository) FindVisibleTo(
	search ProfileSearch,
	options ...repo.Option,
) ([]entity.Profile, error) {

	result := []entity.Profile{}

	err := this.ListQuery(options...).Scopes(visibleScope(search)).Find(&result).Error

	return result, err
}

func (this *ProfileRepository) FindVisibleToCount(
	search ProfileSearch,
	options ...repo.Option,
) (int64, error) {

	count := int64(0)

	err := this.Query(options...).Scopes(visibleScope(search)).Count(&count).Error

	return count, err
}

func (this *ProfileRepository) FindByIdVisibleTo(
	id string,
	organizationId string,
	options ...repo.Option,
) (*entity.Profile, error) {

	query := this.
		ListQuery(options...).
		Scopes(visibleScope(ProfileSearch{OrganizationId: organizationId})).
		Where("profiles.id = ?", id)

	return repo.FirstOrNil[entity.Profile](query)
}

// visibleScope is the single definition of the visibility predicate, shared by the
// three reads so they can never drift apart - a listing that shows a row a lookup
// then refuses is how a scoped profile leaks.
//
// The NULL half cannot be expressed as a typed where: gorm drops a nil pointer from
// the struct condition, so `organization_id IS NULL` would silently disappear and
// every organization would see every row. It is raw SQL for that reason.
func visibleScope(search ProfileSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		if search.Scoped {
			query = query.Where("profiles.organization_id = ?", search.OrganizationId)
		} else {
			query = query.Where(
				"(profiles.organization_id IS NULL OR profiles.organization_id = ?)",
				search.OrganizationId,
			)
		}

		if search.Name != "" {
			query = query.Where("profiles.name ILIKE ?", "%"+search.Name+"%")
		}

		return query
	}
}
