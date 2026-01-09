package services

import (
	"auth_service/app/modules/profile/repository"
	entity "auth_service/infra/entities"
	"encoding/json"
	"math"

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

func (s *ProfileService) GetProfileByAppRole(role string) (*entity.Profile, error) {
	roleProfiles, err := s.profileRepository.FindProfileByAppRole(role)
	if err != nil {
		return nil, err
	}

	var profile entity.Profile
	maxPriority := math.MaxInt

	for _, rp := range roleProfiles {
		permissions := make(map[string]any)

		err := json.Unmarshal(rp.Permission, &permissions)
		if err != nil {
			s.logger.Error("Failed to unmarshal permissions", zap.Error(err))
			continue
		}

		if canRegister, ok := permissions["register"].(bool); ok && canRegister && rp.Priority < maxPriority {
			maxPriority = rp.Priority
			profile = rp.Profile
		}
	}

	return &profile, nil
}
