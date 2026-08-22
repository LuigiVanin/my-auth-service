package mock

import (
	entity "auth_service/infra/entities"
	dto "auth_service/shared/models"

	"github.com/stretchr/testify/mock"
)

// MockSessionService represents a mock implementation of ISessionService
type MockSessionService struct {
	mock.Mock
}

func (this *MockSessionService) CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error) {
	args := this.Called(app, user, request, loginType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionService) EncryptSessionToken(sessionId string, token string, secretKey string) (string, error) {
	args := this.Called(sessionId, token, secretKey)
	return args.String(0), args.Error(1)
}

func (this *MockSessionService) DecryptSessionToken(tokenString string, secretKey string) (string, string, error) {
	args := this.Called(tokenString, secretKey)
	return args.String(0), args.String(1), args.Error(2)
}

func (this *MockSessionService) UseSession(session string) error {
	args := this.Called(session)
	return args.Error(0)
}

// MockSessionRepository represents a mock implementation of ISessionRepository
type MockSessionRepository struct {
	mock.Mock
}

func (this *MockSessionRepository) Create(session entity.Session) (*entity.Session, error) {
	args := this.Called(session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionRepository) FindWhere(where entity.Session, with ...string) (*entity.Session, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionRepository) BatchInvalidateAll(userId uint, appId string, currentSessionId string) error {
	args := this.Called(userId, appId, currentSessionId)
	return args.Error(0)
}

func (this *MockSessionRepository) UpdateLastUsedAt(sessionId string) error {
	args := this.Called(sessionId)
	return args.Error(0)
}
