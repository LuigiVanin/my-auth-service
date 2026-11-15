package mock

import (
	"time"

	dto "auth_service/app/modules/core/invite/models"
	ir "auth_service/app/modules/core/invite/repository"
	is "auth_service/app/modules/core/invite/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

type MockInviteRepository struct {
	mock.Mock
}

var _ ir.IInviteRepository = &MockInviteRepository{}

func (this *MockInviteRepository) FindOne(where entity.Invite, options ...repo.Option) (*entity.Invite, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Invite), args.Error(1)
}

// The first return may also be a func of the arguments, so a test can echo back
// the row the service built.
func (this *MockInviteRepository) Create(invite entity.Invite, options ...repo.Option) (*entity.Invite, error) {
	args := this.Called(invite, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if build, ok := args.Get(0).(func(entity.Invite, ...repo.Option) *entity.Invite); ok {
		return build(invite, options...), args.Error(1)
	}
	return args.Get(0).(*entity.Invite), args.Error(1)
}

func (this *MockInviteRepository) FindForOrganization(organizationId string, status dto.InviteStatus, options ...repo.Option) ([]entity.Invite, error) {
	args := this.Called(organizationId, status, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Invite), args.Error(1)
}

func (this *MockInviteRepository) FindForOrganizationCount(organizationId string, status dto.InviteStatus, options ...repo.Option) (int64, error) {
	args := this.Called(organizationId, status, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockInviteRepository) FindPendingFor(organizationId string, email string, options ...repo.Option) ([]entity.Invite, error) {
	args := this.Called(organizationId, email, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.Invite), args.Error(1)
}

func (this *MockInviteRepository) Accept(id string, at time.Time, options ...repo.Option) (int64, error) {
	args := this.Called(id, at, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockInviteRepository) Revoke(id string, at time.Time, options ...repo.Option) (int64, error) {
	args := this.Called(id, at, options)
	return args.Get(0).(int64), args.Error(1)
}

type MockInviteService struct {
	mock.Mock
}

var _ is.IInviteService = &MockInviteService{}

func (this *MockInviteService) Create(
	organizationId string,
	currentOrganization *entity.Organization,
	caller *entity.Participant,
	inviter *entity.User,
	app *entity.App,
	payload *dto.CreateInvite,
) (*dto.InviteResponse, error) {
	args := this.Called(organizationId, currentOrganization, caller, inviter, app, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.InviteResponse), args.Error(1)
}

func (this *MockInviteService) List(organizationId string, currentOrganization *entity.Organization, query *dto.GetInvitesQuery) (*dto.GetInvitesResponse, error) {
	args := this.Called(organizationId, currentOrganization, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GetInvitesResponse), args.Error(1)
}

func (this *MockInviteService) Revoke(organizationId string, inviteId string, currentOrganization *entity.Organization) (*dto.InviteResponse, error) {
	args := this.Called(organizationId, inviteId, currentOrganization)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.InviteResponse), args.Error(1)
}

func (this *MockInviteService) Resolve(token string, app *entity.App) (*entity.Invite, error) {
	args := this.Called(token, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Invite), args.Error(1)
}

func (this *MockInviteService) Describe(token string, app *entity.App) (*dto.InviteDescription, error) {
	args := this.Called(token, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.InviteDescription), args.Error(1)
}

func (this *MockInviteService) Accept(token string, app *entity.App) (*dto.AcceptInviteResponse, error) {
	args := this.Called(token, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AcceptInviteResponse), args.Error(1)
}

func (this *MockInviteService) AcceptFor(invite *entity.Invite, user *entity.User) (*dto.AcceptInviteResponse, error) {
	args := this.Called(invite, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AcceptInviteResponse), args.Error(1)
}
