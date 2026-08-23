package login_test

import (
	"testing"

	e "auth_service/app/errors"
	"auth_service/app/modules/authorize/services"
	dto "auth_service/app/modules/login/models"
	ls "auth_service/app/modules/login/services"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
	repo "auth_service/shared/repository"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// --- Test Suite ---

type LoginWithPasswordServiceTestSuite struct {
	suite.Suite
	mockUserRepo       *mock.MockUserRepository
	mockHashService    *mock.MockHashService
	mockSessionService *mock.MockSessionService
	mockAuthService    *mock.MockAuthorizeService
	mockOtpService     *mock.MockOtpService

	loginService ls.ILoginService
}

func (this *LoginWithPasswordServiceTestSuite) SetupTest() {
	this.mockUserRepo = new(mock.MockUserRepository)
	this.mockHashService = new(mock.MockHashService)
	this.mockSessionService = new(mock.MockSessionService)
	this.mockAuthService = new(mock.MockAuthorizeService)
	this.mockOtpService = new(mock.MockOtpService)
	logger := zap.NewNop()

	this.loginService = ls.NewLoginService(
		this.mockUserRepo,
		this.mockHashService,
		this.mockSessionService,
		this.mockAuthService,
		this.mockOtpService,
		logger,
	)
}

func (this *LoginWithPasswordServiceTestSuite) TestLoginOnAppWithoutPasswordLoginType_Error() {
	app := &entity.App{
		LoginTypes: []string{"WITH_OTP"},
	}
	userData := dto.LoginPayloadWithPassoword{}
	requestInfo := sharedDto.RequestInfo{}

	response, err := this.loginService.LoginWithPassword(app, userData, requestInfo)

	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowNotAllowed(""), err)

	if ge, ok := err.(*e.AppError); ok {

		assert.Equal(this.T(), 405, ge.Code.Second)
		assert.Equal(
			this.T(),
			e.AppErrorCode("NOT_ALLOWED"),
			ge.Code.First,
		)
	}
}

func (this *LoginWithPasswordServiceTestSuite) TestLoginWithNonExistentUserOnTheApp_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
	}
	payload := dto.LoginPayloadWithPassoword{
		Email:    "nonexistent@example.com",
		Password: "password",
	}
	requestInfo := sharedDto.RequestInfo{}

	// Mock - user not found
	expectedWhere := entity.User{
		Email:       "nonexistent@example.com",
		UsersPoolId: "pool-id",
	}
	this.mockUserRepo.On("FindOne", expectedWhere, []repo.Option{{With: []string{"Profile"}}}).Return(nil, nil) // NOTE: FindOne reports "not found" as nil, nil

	// Act
	response, err := this.loginService.LoginWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
}

func (this *LoginWithPasswordServiceTestSuite) TestLogin_Success() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
	}
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password",
	}
	requestInfo := sharedDto.RequestInfo{}

	user := &entity.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}
	session := &entity.Session{ID: "session-id"}
	credentials := &services.AuthorizationCredentials{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	// Mocks
	expectedWhere := entity.User{
		Email:       "test@example.com",
		UsersPoolId: "pool-id",
	}
	this.mockUserRepo.On("FindOne", expectedWhere, []repo.Option{{With: []string{"Profile"}}}).Return(user, nil)
	this.mockHashService.On("Compare", "password", "hashed_password").Return(true, nil)
	this.mockSessionService.On("CreateNew", app, user, requestInfo, "WITH_PASSWORD").Return(session, nil)
	this.mockAuthService.On("CreateAuthorizationCredentials", app, session).Return(credentials, nil)

	// Act
	response, err := this.loginService.LoginWithPassword(app, payload, requestInfo)

	// Assert
	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), response)
	assert.Equal(this.T(), "session-id", response.SessionId)
	assert.Equal(this.T(), "access-token", response.AccessToken)
}

func (this *LoginWithPasswordServiceTestSuite) TestLoginWithPassword_InvalidPassword_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
	}
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	requestInfo := sharedDto.RequestInfo{}

	user := &entity.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	// Mocks
	expectedWhere := entity.User{
		Email:       "test@example.com",
		UsersPoolId: "pool-id",
	}
	this.mockUserRepo.On("FindOne", expectedWhere, []repo.Option{{With: []string{"Profile"}}}).Return(user, nil)
	this.mockHashService.On("Compare", "wrongpassword", "hashed_password").Return(false, nil)

	// Act
	response, err := this.loginService.LoginWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowUnauthorizedError(""), err)
}

func (this *LoginWithPasswordServiceTestSuite) TestLoginWithPassword_HashServiceFails_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
	}
	payload := dto.LoginPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password",
	}
	requestInfo := sharedDto.RequestInfo{}

	user := &entity.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	// Mocks
	expectedWhere := entity.User{
		Email:       "test@example.com",
		UsersPoolId: "pool-id",
	}
	this.mockUserRepo.On("FindOne", expectedWhere, []repo.Option{{With: []string{"Profile"}}}).Return(user, nil)
	this.mockHashService.On("Compare", "password", "hashed_password").Return(false, assert.AnError)

	// Act
	response, err := this.loginService.LoginWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowInternalServerError(""), err)
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(LoginWithPasswordServiceTestSuite))
}
