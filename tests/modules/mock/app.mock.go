package mock

import (
	dto "auth_service/app/modules/core/app/models"
	ar "auth_service/app/modules/core/app/repository"
	as "auth_service/app/modules/core/app/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockAppRepository represents a mock implementation of IAppRepository
type MockAppRepository struct {
	mock.Mock
}

var _ ar.IAppRepository = &MockAppRepository{}

func (this *MockAppRepository) FindOne(where entity.App, options ...repo.Option) (*entity.App, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) Create(app entity.App, options ...repo.Option) (*entity.App, error) {
	args := this.Called(app, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) Update(where entity.App, data dto.AppUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockAppRepository) FindSearch(search ar.AppSearch, options ...repo.Option) ([]entity.App, error) {
	args := this.Called(search, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.App), args.Error(1)
}

func (this *MockAppRepository) FindSearchCount(search ar.AppSearch, options ...repo.Option) (int64, error) {
	args := this.Called(search, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockAppService represents a mock implementation of IAppService
type MockAppService struct {
	mock.Mock
}

var _ as.IAppService = &MockAppService{}

func (this *MockAppService) CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, currentOrganization *entity.Organization, participant *entity.Participant, payload *dto.CreateAppPayload) (*entity.App, error) {
	args := this.Called(currentUser, currentApp, currentOrganization, participant, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppService) FindAll(currentUser *entity.User, currentApp *entity.App, currentOrganization *entity.Organization, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error) {
	args := this.Called(currentUser, currentApp, currentOrganization, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetAppsResponse), args.Error(1)
}

func (this *MockAppService) FindById(id string, currentOrganization *entity.Organization) (*entity.App, error) {
	args := this.Called(id, currentOrganization)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppService) FindAllUserApps(userId string, currentOrganization *entity.Organization) ([]entity.App, error) {
	args := this.Called(userId, currentOrganization)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.App), args.Error(1)
}

func (this *MockAppService) Update(user *entity.User, app *entity.App, currentOrganization *entity.Organization) (*entity.App, error) {
	args := this.Called(user, app, currentOrganization)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}
