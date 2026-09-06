package mock

import (
	dto "auth_service/app/modules/core/participant/models"
	pr "auth_service/app/modules/core/participant/repository"
	ps "auth_service/app/modules/core/participant/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockParticipantRepository represents a mock implementation of IParticipantRepository
type MockParticipantRepository struct {
	mock.Mock
}

var _ pr.IParticipantRepository = &MockParticipantRepository{}

func (this *MockParticipantRepository) FindOne(where entity.Participant, options ...repo.Option) (*entity.Participant, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Participant), args.Error(1)
}

func (this *MockParticipantRepository) Create(participant entity.Participant, options ...repo.Option) (*entity.Participant, error) {
	args := this.Called(participant, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Participant), args.Error(1)
}

func (this *MockParticipantRepository) Update(where entity.Participant, data dto.ParticipantUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockParticipantRepository) Delete(where entity.Participant, options ...repo.Option) (int64, error) {
	args := this.Called(where, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockParticipantRepository) FindByOrganizationAndUser(organizationId string, userId uint, options ...repo.Option) (*entity.Participant, error) {
	args := this.Called(organizationId, userId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Participant), args.Error(1)
}

func (this *MockParticipantRepository) FindByOrganizationId(organizationId string, options ...repo.Option) ([]entity.Participant, error) {
	args := this.Called(organizationId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Participant), args.Error(1)
}

func (this *MockParticipantRepository) FindByOrganizationIdCount(organizationId string, options ...repo.Option) (int64, error) {
	args := this.Called(organizationId, options)
	return args.Get(0).(int64), args.Error(1)
}

// MockParticipantService represents a mock implementation of IParticipantService
type MockParticipantService struct {
	mock.Mock
}

var _ ps.IParticipantService = &MockParticipantService{}

func (this *MockParticipantService) Create(organizationId string, userId uint, profileId string) (*entity.Participant, error) {
	args := this.Called(organizationId, userId, profileId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Participant), args.Error(1)
}

func (this *MockParticipantService) FindForUserInOrganization(organizationId string, userId uint) (*entity.Participant, error) {
	args := this.Called(organizationId, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Participant), args.Error(1)
}

func (this *MockParticipantService) FindAllInOrganization(
	organizationId string,
	requestingUserId uint,
	query *dto.GetParticipantsQuery,
) (*dto.GetParticipantsResponse, error) {
	args := this.Called(organizationId, requestingUserId, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetParticipantsResponse), args.Error(1)
}

func (this *MockParticipantService) FindForCurrentOrganization(user *entity.User) (*ps.ResolvedParticipation, error) {
	args := this.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ps.ResolvedParticipation), args.Error(1)
}
