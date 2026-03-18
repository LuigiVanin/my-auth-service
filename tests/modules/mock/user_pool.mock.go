package mock

import (
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

type MockUserPoolRepository struct {
	mock.Mock
}

func (this *MockUserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UsersPool, error) {
	args := this.Called(id, appId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolRepository) Create(userPool *entity.UsersPool) (*entity.UsersPool, error) {
	args := this.Called(userPool)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolRepository) FindById(id string) (*entity.UsersPool, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}
