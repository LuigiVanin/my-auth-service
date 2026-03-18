package mock

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

type MockRegisterService struct {
	mock.Mock
}

func (this *MockRegisterService) Register() error {
	args := this.Called()
	return args.Error(0)
}

func (this *MockRegisterService) RegisterWithPassword(
	app *entity.App,
	userData dto.RegisterPayloadWithPassoword,
	request dto.RequestInfo,
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
	request dto.RequestInfo,
) (*dto.RegisterResponse, error) {
	args := this.Called(app, payload, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RegisterResponse), args.Error(1)
}
