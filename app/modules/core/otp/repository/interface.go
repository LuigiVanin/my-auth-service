package repository

import (
	dto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IOtpRepository interface {
	FindOne(where entity.Otp, options ...repo.Option) (*entity.Otp, error)
	Create(otp entity.Otp, options ...repo.Option) (*entity.Otp, error)
	Update(where entity.Otp, data dto.OtpUpdateDao, options ...repo.Option) (int64, error)

	// Dedicated query: "the most recent one" is an ordering decision that
	// belongs to the repository, not to the caller.
	FindLast(where entity.Otp, options ...repo.Option) (*entity.Otp, error)
}
