package mock

import (
	dto "auth_service/app/modules/core/user_pool/models"
	upr "auth_service/app/modules/core/user_pool/repository"
	ups "auth_service/app/modules/core/user_pool/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockUserPoolRepository represents a mock implementation of IUserPoolRepository
type MockUserPoolRepository struct {
	mock.Mock
}

var _ upr.IUserPoolRepository = &MockUserPoolRepository{}

func (this *MockUserPoolRepository) FindOne(where entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolRepository) Create(userPool entity.UsersPool, options ...repo.Option) (*entity.UsersPool, error) {
	args := this.Called(userPool, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolRepository) Update(where entity.UsersPool, data dto.UserPoolUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockUserPoolService represents a mock implementation of IUserPoolService
type MockUserPoolService struct {
	mock.Mock
}

var _ ups.IUserPoolService = &MockUserPoolService{}

func (this *MockUserPoolService) Create(data ups.CreateUserPoolData, granter *entity.Organization) (*entity.UsersPool, error) {
	args := this.Called(data, granter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolService) FindById(id string) (*entity.UsersPool, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}

func (this *MockUserPoolService) FindByIdInOrganization(id string, organizationId string) (*entity.UsersPool, error) {
	args := this.Called(id, organizationId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UsersPool), args.Error(1)
}
