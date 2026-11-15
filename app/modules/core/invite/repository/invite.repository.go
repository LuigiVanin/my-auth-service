package repository

import (
	"fmt"
	"time"

	dto "auth_service/app/modules/core/invite/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InviteRepository struct {
	repo.BaseRepository[entity.Invite, repo.NoUpdate]
}

var _ IInviteRepository = &InviteRepository{}

func NewInviteRepository(client *gorm.DB) *InviteRepository {
	return &InviteRepository{
		BaseRepository: repo.NewBaseRepository[entity.Invite, repo.NoUpdate](client),
	}
}

// Raw SQL and never a struct condition: gorm drops the nil pointers and the false
// a struct would carry, and the predicate would vanish in silence.
const pendingPredicate = "accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()"

var statusPredicates = map[dto.InviteStatus]string{
	dto.InviteStatusPending:  pendingPredicate,
	dto.InviteStatusAccepted: "accepted_at IS NOT NULL",
	dto.InviteStatusRevoked:  "revoked_at IS NOT NULL",
	dto.InviteStatusExpired:  "accepted_at IS NULL AND revoked_at IS NULL AND expires_at <= now()",
}

func (this *InviteRepository) forOrganization(query *gorm.DB, organizationId string, status dto.InviteStatus) (*gorm.DB, error) {
	query = query.Where("organization_id = ?", organizationId)

	if status == "" {
		return query, nil
	}

	predicate, ok := statusPredicates[status]

	if !ok {
		return nil, fmt.Errorf("unknown invite status %q", status)
	}

	return query.Where(predicate), nil
}

func (this *InviteRepository) FindForOrganization(
	organizationId string,
	status dto.InviteStatus,
	options ...repo.Option,
) ([]entity.Invite, error) {
	result := []entity.Invite{}

	query, err := this.forOrganization(this.ListQuery(options...), organizationId, status)

	if err != nil {
		return nil, err
	}

	err = query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).
		Find(&result).Error

	return result, err
}

func (this *InviteRepository) FindForOrganizationCount(
	organizationId string,
	status dto.InviteStatus,
	options ...repo.Option,
) (int64, error) {
	var count int64

	query, err := this.forOrganization(this.Query(options...), organizationId, status)

	if err != nil {
		return 0, err
	}

	err = query.Count(&count).Error

	return count, err
}

// Unpaginated on purpose: every pending invite for the address is revoked when a
// new one replaces them, and a page would leave the rest alive.
func (this *InviteRepository) FindPendingFor(organizationId string, email string, options ...repo.Option) ([]entity.Invite, error) {
	result := []entity.Invite{}

	err := this.Query(options...).
		Where("organization_id = ? AND email = ?", organizationId, email).
		Where(pendingPredicate).
		Find(&result).Error

	return result, err
}

func (this *InviteRepository) Accept(id string, at time.Time, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		Where(pendingPredicate).
		Update("accepted_at", at)

	return result.RowsAffected, result.Error
}

func (this *InviteRepository) Revoke(id string, at time.Time, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		Where(pendingPredicate).
		Update("revoked_at", at)

	return result.RowsAffected, result.Error
}
