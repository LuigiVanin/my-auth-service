package mock

import (
	pr "auth_service/app/modules/core/profile/repository"
	ps "auth_service/app/modules/core/profile/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockProfileRepository represents a mock implementation of IProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

var _ pr.IProfileRepository = &MockProfileRepository{}

func (this *MockProfileRepository) FindOne(where entity.Profile, options ...repo.Option) (*entity.Profile, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileRepository) FindByKey(key string, options ...repo.Option) (*entity.Profile, error) {
	args := this.Called(key, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

// MockProfileService represents a mock implementation of IProfileService
type MockProfileService struct {
	mock.Mock
}

var _ ps.IProfileService = &MockProfileService{}

func (this *MockProfileService) FindById(id string) (*entity.Profile, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileService) FindByKey(key string) (*entity.Profile, error) {
	args := this.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}
