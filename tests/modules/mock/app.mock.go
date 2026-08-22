package mock

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

// MockAppRepository represents a mock implementation of IAppRepository
type MockAppRepository struct {
	mock.Mock
}

func (this *MockAppRepository) FindAppbyIdWithPool(id string) (*entity.App, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) Create(app *entity.App) (*entity.App, error) {
	args := this.Called(app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) FindWhere(where entity.App, with ...string) (*entity.App, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) FindManyWhereAndCount(
	where any,
	args []any,
	skip int,
	limit int,
	with ...string,
) ([]entity.App, int64, error) {
	mockArgs := this.Called(where, args, skip, limit, with)
	if mockArgs.Get(0) == nil {
		return nil, 0, mockArgs.Error(2)
	}
	return mockArgs.Get(0).([]entity.App), mockArgs.Get(1).(int64), mockArgs.Error(2)
}

func (this *MockAppRepository) Update(id string, app entity.App) (*entity.App, error) {
	args := this.Called(id, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

// MockAppService represents a mock implementation of IAppService
type MockAppService struct {
	mock.Mock
}

func (this *MockAppService) CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, payload *dto.CreateAppPayload) (*entity.App, error) {
	args := this.Called(currentUser, currentApp, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppService) FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error) {
	args := this.Called(currentUser, currentApp, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetAppsResponse), args.Error(1)
}

func (this *MockAppService) FindById(id string) (*entity.App, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppService) FindAllUserApps(userId string) ([]entity.App, error) {
	args := this.Called(userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.App), args.Error(1)
}

func (this *MockAppService) Update(user *entity.User, app *entity.App) (*entity.App, error) {
	args := this.Called(user, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}
