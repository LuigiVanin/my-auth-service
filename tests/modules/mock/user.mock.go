package mock

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository represents a mock implementation of IUserRepository
type MockUserRepository struct {
	mock.Mock
}

func (this *MockUserRepository) FindWhere(where entity.User, with ...string) (*entity.User, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) FindManyWhere(where entity.User, skip int, limit int, with ...string) (*[]entity.User, int64, error) {
	args := this.Called(where, skip, limit, with)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).(*[]entity.User), args.Get(1).(int64), args.Error(2)
}

func (this *MockUserRepository) Create(user entity.User) (*entity.User, error) {
	args := this.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) Update(user *entity.User) error {
	args := this.Called(user)
	return args.Error(0)
}

func (this *MockUserRepository) FindFromAppId(id string, skip int, limit int, with ...string) ([]entity.User, int64, error) {
	args := this.Called(id, skip, limit, with)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]entity.User), args.Get(1).(int64), args.Error(2)
}

// MockUserService represents a mock implementation of IUserService
type MockUserService struct {
	mock.Mock
}

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

func (this *MockUserService) Update(user *entity.User) error {
	args := this.Called(user)
	return args.Error(0)
}

func (this *MockUserService) FindAllUsersFromApp(targetAppId string, currentUser *entity.User, targetApp *entity.App, query *dto.GetUsersAppQuery) (*dto.GetUsersAppResponse, error) {
	args := this.Called(targetAppId, currentUser, targetApp, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetUsersAppResponse), args.Error(1)
}
