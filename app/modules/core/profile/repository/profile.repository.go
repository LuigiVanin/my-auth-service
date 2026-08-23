package repository

import (
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NOTE: app_role_profiles is a read only reference table, hence repo.NoUpdate.
type ProfileRepository struct {
	repo.BaseRepository[entity.AppRoleProfile, repo.NoUpdate]
}

var _ IProfileRepository = &ProfileRepository{}

func NewProfileRepository(client *gorm.DB) *ProfileRepository {
	return &ProfileRepository{
		BaseRepository: repo.NewBaseRepository[entity.AppRoleProfile, repo.NoUpdate](client),
	}
}

// FindByAppRole returns every profile bound to the role, lowest priority first.
//
// NOTE: this builds on Query, not ListQuery, so the default page size is not
// applied. The caller resolves a single winning profile out of the whole set,
// so a truncated result would silently pick the wrong one. It is safe here
// because the table holds a handful of rows per role.
func (this *ProfileRepository) FindByAppRole(role string, options ...repo.Option) ([]entity.AppRoleProfile, error) {
	result := []entity.AppRoleProfile{}

	err := this.Query(options...).
		Preload("Profile").
		Where("role = ?", role).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "priority"}}).
		Find(&result).Error

	return result, err
}
