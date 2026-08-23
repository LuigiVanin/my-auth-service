package repository

import (
	dto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OtpRepository struct {
	repo.BaseRepository[entity.Otp, dto.OtpUpdateDao]
}

var _ IOtpRepository = &OtpRepository{}

func NewOtpRepository(client *gorm.DB) *OtpRepository {
	return &OtpRepository{
		BaseRepository: repo.NewBaseRepository[entity.Otp, dto.OtpUpdateDao](client),
	}
}

// FindLast returns the most recent OTP matching where, or nil when there is
// none. Sorting by created_at is part of the query and is never left to the
// caller, otherwise a caller that forgets it silently gets an arbitrary row.
func (this *OtpRepository) FindLast(where entity.Otp, options ...repo.Option) (*entity.Otp, error) {
	query := this.ListQuery(options...).
		Where(where).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true})

	return repo.FirstOrNil[entity.Otp](query)
}
