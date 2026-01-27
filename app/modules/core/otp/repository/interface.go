package repository

import (
	entity "auth_service/infra/entities"
)

type IOtpRepository interface {
	Create(otp *entity.Otp) (*entity.Otp, error)
	FindById(otpId string) (*entity.Otp, error)
	FindLastOneWhere(where entity.Otp, with ...string) (*entity.Otp, error)

	Invalidate(otpId string) error
}
