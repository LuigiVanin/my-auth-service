package mock

import (
	dto "auth_service/app/modules/register/models"
	rs "auth_service/app/modules/register/services"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"

	"github.com/stretchr/testify/mock"
)

type MockRegisterService struct {
	mock.Mock
}

func (this *MockRegisterService) ProvisionUser(app *entity.App, user entity.User) (*rs.ProvisionedUser, error) {
	args := this.Called(app, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rs.ProvisionedUser), args.Error(1)
}

func (this *MockRegisterService) Register() error {
	args := this.Called()
	return args.Error(0)
}

func (this *MockRegisterService) RegisterWithPassword(
	app *entity.App,
	userData dto.RegisterPayloadWithPassoword,
	request sharedDto.RequestInfo,
) (*dto.RegisterResponse, error) {
	args := this.Called(app, userData, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RegisterResponse), args.Error(1)
}

func (this *MockRegisterService) RegisterWithOtp(
	app *entity.App,
	payload dto.RegisterPayloadWithOtp,
	request sharedDto.RequestInfo,
) (*dto.RegisterResponse, error) {
	args := this.Called(app, payload, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RegisterResponse), args.Error(1)
}
