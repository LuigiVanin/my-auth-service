package mock

import (
	dto "auth_service/app/modules/core/user/models"
	ur "auth_service/app/modules/core/user/repository"
	us "auth_service/app/modules/core/user/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository represents a mock implementation of IUserRepository
type MockUserRepository struct {
	mock.Mock
}

var _ ur.IUserRepository = &MockUserRepository{}

func (this *MockUserRepository) Find(where entity.User, options ...repo.Option) ([]entity.User, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.User), args.Error(1)
}

func (this *MockUserRepository) FindCount(where entity.User, options ...repo.Option) (int64, error) {
	args := this.Called(where, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockUserRepository) FindOne(where entity.User, options ...repo.Option) (*entity.User, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) Create(user entity.User, options ...repo.Option) (*entity.User, error) {
	args := this.Called(user, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) Update(where entity.User, data dto.UserUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockUserRepository) Delete(where entity.User, options ...repo.Option) (int64, error) {
	args := this.Called(where, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockUserRepository) FindFromAppId(appId string, options ...repo.Option) ([]entity.User, error) {
	args := this.Called(appId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.User), args.Error(1)
}

func (this *MockUserRepository) FindFromAppIdCount(appId string, options ...repo.Option) (int64, error) {
	args := this.Called(appId, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockUserService represents a mock implementation of IUserService
type MockUserService struct {
	mock.Mock
}

var _ us.IUserService = &MockUserService{}

func (this *MockUserService) IsAlreadyCreated(email string, app *entity.App) (bool, error) {
	args := this.Called(email, app)
	return args.Bool(0), args.Error(1)
}

func (this *MockUserService) FindUserInPool(email string, usersPoolId string) (*entity.User, error) {
	args := this.Called(email, usersPoolId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserService) Update(where entity.User, data dto.UserUpdateDao) (int64, error) {
	args := this.Called(where, data)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockUserService) FindAllUsersFromApp(targetAppId string, currentUser *entity.User, targetApp *entity.App, currentOrganization *entity.Organization, query *dto.GetUsersAppQuery) (*dto.GetUsersAppResponse, error) {
	args := this.Called(targetAppId, currentUser, targetApp, currentOrganization, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetUsersAppResponse), args.Error(1)
}
