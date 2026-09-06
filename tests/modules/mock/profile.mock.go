package mock

import (
	dto "auth_service/app/modules/core/profile/models"
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

func (this *MockProfileRepository) FindVisibleTo(search pr.ProfileSearch, options ...repo.Option) ([]entity.Profile, error) {
	args := this.Called(search, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Profile), args.Error(1)
}

func (this *MockProfileRepository) FindVisibleToCount(search pr.ProfileSearch, options ...repo.Option) (int64, error) {
	args := this.Called(search, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockProfileRepository) FindByIdVisibleTo(id string, organizationId string, options ...repo.Option) (*entity.Profile, error) {
	args := this.Called(id, organizationId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileRepository) Create(value entity.Profile, options ...repo.Option) (*entity.Profile, error) {
	args := this.Called(value, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileRepository) Update(where entity.Profile, dao dto.ProfileUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, dao, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockProfileService represents a mock implementation of IProfileService
type MockProfileService struct {
	mock.Mock
}

var _ ps.IProfileService = &MockProfileService{}

func (this *MockProfileService) FindByIdVisibleTo(id string, organizationId string) (*entity.Profile, error) {
	args := this.Called(id, organizationId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileService) FindAllVisibleTo(organizationId string, query *dto.GetProfilesQuery) (*dto.GetProfilesResponse, error) {
	args := this.Called(organizationId, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetProfilesResponse), args.Error(1)
}

func (this *MockProfileService) CreateForOrganization(caller *ps.ProfileWriteContext, payload *dto.CreateProfile) (*entity.Profile, error) {
	args := this.Called(caller, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Profile), args.Error(1)
}

func (this *MockProfileService) UpdateForOrganization(id string, caller *ps.ProfileWriteContext, payload *dto.UpdateProfile) (*entity.Profile, error) {
	args := this.Called(id, caller, payload)
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
