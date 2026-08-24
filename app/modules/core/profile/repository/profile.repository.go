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
