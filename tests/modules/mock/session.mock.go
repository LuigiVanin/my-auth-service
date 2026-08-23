package mock

import (
	sdto "auth_service/app/modules/core/session/models"
	sr "auth_service/app/modules/core/session/repository"
	ss "auth_service/app/modules/core/session/services"
	entity "auth_service/infra/entities"
	dto "auth_service/shared/models"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockSessionService represents a mock implementation of ISessionService
type MockSessionService struct {
	mock.Mock
}

var _ ss.ISessionService = &MockSessionService{}

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

var _ sr.ISessionRepository = &MockSessionRepository{}

func (this *MockSessionRepository) FindOne(where entity.Session, options ...repo.Option) (*entity.Session, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionRepository) Create(session entity.Session, options ...repo.Option) (*entity.Session, error) {
	args := this.Called(session, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionRepository) Update(where entity.Session, data sdto.SessionUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockSessionRepository) FindActive(sessionId string, options ...repo.Option) (*entity.Session, error) {
	args := this.Called(sessionId, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionRepository) InvalidateAllExcept(userId uint, appId string, currentSessionId string, options ...repo.Option) (int64, error) {
	args := this.Called(userId, appId, currentSessionId, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockSessionRepository) TouchLastUsedAt(sessionId string, options ...repo.Option) (int64, error) {
	args := this.Called(sessionId, options)
	return args.Get(0).(int64), args.Error(1)
}
