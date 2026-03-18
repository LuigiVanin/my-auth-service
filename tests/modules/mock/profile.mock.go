package mock

import (
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

// MockProfileService represents a mock implementation of IProfileService
type MockProfileService struct {
	mock.Mock
}

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

func (this *MockProfileRepository) FindProfileByAppRole(role string) ([]entity.AppRoleProfile, error) {
	args := this.Called(role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.AppRoleProfile), args.Error(1)
}
