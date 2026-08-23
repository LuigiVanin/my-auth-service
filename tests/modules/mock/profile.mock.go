package mock

import (
	pr "auth_service/app/modules/core/profile/repository"
	ps "auth_service/app/modules/core/profile/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockProfileService represents a mock implementation of IProfileService
type MockProfileService struct {
	mock.Mock
}

var _ ps.IProfileService = &MockProfileService{}

func (this *MockProfileService) GetProfileByAppRole(role string) (*entity.Profile, error) {
	args := this.Called(role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

// MockProfileRepository represents a mock implementation of IProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

var _ pr.IProfileRepository = &MockProfileRepository{}

func (this *MockProfileRepository) FindByAppRole(role string, options ...repo.Option) ([]entity.AppRoleProfile, error) {
	args := this.Called(role, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.AppRoleProfile), args.Error(1)
}
