package repository

import (
	entity "auth_service/infra/entities"
	"errors"

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

func (repo *OtpRepository) FindById(otpId string) (*entity.Otp, error) {
	var otp entity.Otp

	err := repo.db.Where("id = ?", otpId).First(&otp).Error
	if err != nil {
		return nil, err
	}

	return &otp, nil
}

func (repo *OtpRepository) FindLastOneWhere(where entity.Otp, with ...string) (*entity.Otp, error) {
	var result entity.Otp

	// NOTE: Sorting by created_at in descending order to get the latest OTP ALWAYS
	whereClause := repo.db.Where(where).Order("created_at DESC")

	if len(with) > 0 {
		for _, relation := range with {
			whereClause = whereClause.Preload(relation)
		}
	}

	err := whereClause.First(&result).Error
	return &result, err
}

func (repo *OtpRepository) Update(otp *entity.Otp) error {
	if otp.ID == "" {
		return errors.New("otp id is required")
	}

	return repo.db.Save(otp).Error
}

func (repo *OtpRepository) Invalidate(otpId string) error {
	err := repo.db.Model(&entity.Otp{}).Where("id = ?", otpId).Update("invalidated", true).Error
	if err != nil {
		return err
	}
	return nil
}
