package mock

import (
	dto "auth_service/app/modules/core/organization/models"
	or "auth_service/app/modules/core/organization/repository"
	os "auth_service/app/modules/core/organization/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockOrganizationRepository represents a mock implementation of IOrganizationRepository
type MockOrganizationRepository struct {
	mock.Mock
}

var _ or.IOrganizationRepository = &MockOrganizationRepository{}

func (this *MockOrganizationRepository) FindOne(where entity.Organization, options ...repo.Option) (*entity.Organization, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}

func (this *MockOrganizationRepository) Create(organization entity.Organization, options ...repo.Option) (*entity.Organization, error) {
	args := this.Called(organization, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}

func (this *MockOrganizationRepository) Update(where entity.Organization, data dto.OrganizationUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockOrganizationRepository) FindByParticipantUserId(userId uint, options ...repo.Option) ([]entity.Organization, error) {
	args := this.Called(userId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Organization), args.Error(1)
}

func (this *MockOrganizationRepository) FindByParticipantUserIdCount(userId uint, options ...repo.Option) (int64, error) {
	args := this.Called(userId, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockOrganizationService represents a mock implementation of IOrganizationService
type MockOrganizationService struct {
	mock.Mock
}

var _ os.IOrganizationService = &MockOrganizationService{}

func (this *MockOrganizationService) Create(data dto.CreateOrganizationData) (*entity.Organization, error) {
	args := this.Called(data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}

func (this *MockOrganizationService) SetOwner(organizationId string, ownerUserId uint) error {
	args := this.Called(organizationId, ownerUserId)
	return args.Error(0)
}

func (this *MockOrganizationService) FindById(id string) (*entity.Organization, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}

func (this *MockOrganizationService) FindAllForUser(userId uint, query *dto.GetOrganizationsQuery) (*dto.GetOrganizationsResponse, error) {
	args := this.Called(userId, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetOrganizationsResponse), args.Error(1)
}

func (this *MockOrganizationService) Switch(user *entity.User, organizationId string) (*entity.Organization, error) {
	args := this.Called(user, organizationId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}

func (this *MockOrganizationService) CreateForUser(
	currentUser *entity.User,
	currentOrganization *entity.Organization,
	payload *dto.CreateOrganizationPayload,
) (*entity.Organization, error) {
	args := this.Called(currentUser, currentOrganization, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Organization), args.Error(1)
}
