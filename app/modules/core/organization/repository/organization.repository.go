package repository

import (
	dto "auth_service/app/modules/core/organization/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type OrganizationRepository struct {
	repo.BaseRepository[entity.Organization, dto.OrganizationUpdateDao]
}

var _ IOrganizationRepository = &OrganizationRepository{}

func NewOrganizationRepository(client *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{
		BaseRepository: repo.NewBaseRepository[entity.Organization, dto.OrganizationUpdateDao](client),
	}
}

// Shared by the listing and the count so the two can never drift.
const organizationByParticipantJoin = "JOIN participants ON participants.organization_id = organizations.id"

func (this *OrganizationRepository) FindByParticipantUserId(
	userId uint,
	options ...repo.Option,
) ([]entity.Organization, error) {
	result := []entity.Organization{}

	err := this.ListQuery(options...).
		Joins(organizationByParticipantJoin).
		Where("participants.user_id = ?", userId).
		Find(&result).Error

	return result, err
}

// NOTE: no Distinct. A count over a join normally over counts, which is the trap
// steering/repository-pattern.md warns about. It is safe only because of the
// unique index on participants (organization_id, user_id).
func (this *OrganizationRepository) FindByParticipantUserIdCount(
	userId uint,
	options ...repo.Option,
) (int64, error) {
	var count int64

	err := this.Query(options...).
		Joins(organizationByParticipantJoin).
		Where("participants.user_id = ?", userId).
		Count(&count).Error

	return count, err
}

// WriteAvatarKey is the only writer of avatar_key: the column is absent from the
// update dao. nil clears it. Update and not UpdateColumn, so the change moves
// `updated_at`.
func (this *OrganizationRepository) WriteAvatarKey(id string, key *string, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		Update("avatar_key", key)

	return result.RowsAffected, result.Error
}
