package services

import (
	e "auth_service/app/errors"
	"auth_service/app/modules/core/profile/repository"
	entity "auth_service/infra/entities"

	"go.uber.org/zap"
)

type ProfileService struct {
	profileRepository repository.IProfileRepository
	logger            *zap.Logger
}

var _ IProfileService = &ProfileService{}

func NewProfileService(profileRepository repository.IProfileRepository, logger *zap.Logger) *ProfileService {
	return &ProfileService{
		profileRepository: profileRepository,
		logger:            logger,
	}
}

func (this *ProfileService) FindById(id string) (*entity.Profile, error) {
	profile, err := this.profileRepository.FindOne(entity.Profile{ID: id})

	if err != nil {
		this.logger.Error("Failed to look up a profile by id", zap.String("id", id), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	return profile, nil
}

func (this *ProfileService) FindByKey(key string) (*entity.Profile, error) {
	profile, err := this.profileRepository.FindByKey(key)

	if err != nil {
		this.logger.Error("Failed to look up a profile by key", zap.String("key", key), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	return profile, nil
}
