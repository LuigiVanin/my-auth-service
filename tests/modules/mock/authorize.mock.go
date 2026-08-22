package mock

import (
	dto "auth_service/app/modules/authorize/models"
	"auth_service/app/modules/authorize/services"
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

type MockAuthorizeService struct {
	mock.Mock
}

func (this *MockAuthorizeService) Authorize(app *entity.App, token string, ip string) (*dto.AuthorizeResponse, error) {
	args := this.Called(app, token, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuthorizeResponse), args.Error(1)
}

func (this *MockAuthorizeService) Refresh(app *entity.App, token string, ip string) (*dto.RefreshResponse, error) {
	args := this.Called(app, token, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RefreshResponse), args.Error(1)
}

func (this *MockAuthorizeService) CreateAuthorizationCredentials(app *entity.App, session *entity.Session) (*services.AuthorizationCredentials, error) {
	args := this.Called(app, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthorizationCredentials), args.Error(1)
}

func (this *MockAuthorizeService) ResetPassword(app *entity.App, payload dto.ResetPasswordPayload) (*entity.User, error) {
	args := this.Called(app, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
