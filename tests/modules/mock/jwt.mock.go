package mock

import (
	"auth_service/app/models/dto"

	"github.com/stretchr/testify/mock"
)

type MockJwtService struct {
	mock.Mock
}

func (this *MockJwtService) CreateAuthToken(payload dto.AuthPayload, key string) (string, error) {
	args := this.Called(payload, key)
	return args.String(0), args.Error(1)
}

func (this *MockJwtService) ParseAuthToken(jwt string, key string) (*dto.AuthPayload, error) {
	args := this.Called(jwt, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuthPayload), args.Error(1)
}
