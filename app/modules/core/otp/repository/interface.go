package repository

import (
	entity "auth_service/infra/entities"
)

type IOtpRepository interface {
	Create(otp *entity.Otp) (*entity.Otp, error)
}
