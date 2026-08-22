package mock

import (
	dto "auth_service/app/modules/login/models"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"

	"github.com/stretchr/testify/mock"
)

type MockLoginService struct {
	mock.Mock
}

func (this *MockLoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword, request sharedDto.RequestInfo) (*dto.LoginResponse, error) {
	args := this.Called(app, userData, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResponse), args.Error(1)
}

func (this *MockLoginService) LoginWithOtp(app *entity.App, userData dto.LoginPayloadWithOtp, request sharedDto.RequestInfo) (*dto.LoginResponse, error) {
	args := this.Called(app, userData, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResponse), args.Error(1)
}
