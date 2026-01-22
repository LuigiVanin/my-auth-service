package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type OtpRepository struct {
	db *gorm.DB
}

func NewOtpRepository(db *gorm.DB) *OtpRepository {
	return &OtpRepository{
		db: db,
	}
}

func (repo *OtpRepository) Create(otp *entity.Otp) (*entity.Otp, error) {
	result := repo.db.Create(otp)
	if result.Error != nil {
		return nil, result.Error
	}
	return otp, nil
}
