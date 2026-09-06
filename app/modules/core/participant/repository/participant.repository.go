package repository

import (
	dto "auth_service/app/modules/core/participant/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type ParticipantRepository struct {
	repo.BaseRepository[entity.Participant, dto.ParticipantUpdateDao]
}

var _ IParticipantRepository = &ParticipantRepository{}

func NewParticipantRepository(client *gorm.DB) *ParticipantRepository {
	return &ParticipantRepository{
		BaseRepository: repo.NewBaseRepository[entity.Participant, dto.ParticipantUpdateDao](client),
	}
}

// NOTE: UserId is a uint and gorm drops zero valued fields from a struct
// condition, so a typed where would compile to `WHERE organization_id = $1` and
// hand back the first participant of the organization, whoever it is.
func (this *ParticipantRepository) FindByOrganizationAndUser(
	organizationId string,
	userId uint,
	options ...repo.Option,
) (*entity.Participant, error) {
	query := this.ListQuery(options...).
		Where("organization_id = ? AND user_id = ?", organizationId, userId)

	return repo.FirstOrNil[entity.Participant](query)
}

func (this *ParticipantRepository) FindByOrganizationId(
	organizationId string,
	options ...repo.Option,
) ([]entity.Participant, error) {
	result := []entity.Participant{}

	err := this.ListQuery(options...).
		Where("organization_id = ?", organizationId).
		Find(&result).Error

	return result, err
}

func (this *ParticipantRepository) FindByOrganizationIdCount(
	organizationId string,
	options ...repo.Option,
) (int64, error) {
	var count int64

	err := this.Query(options...).
		Where("organization_id = ?", organizationId).
		Count(&count).Error

	return count, err
}
