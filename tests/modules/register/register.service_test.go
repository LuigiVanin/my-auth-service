package register_test

import (
	"encoding/json"
	"testing"

	e "auth_service/app/errors"
	"auth_service/app/models/dto"
	"auth_service/app/modules/authorize/services"
	rs "auth_service/app/modules/register/services"
	entity "auth_service/infra/entities"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// --- Test Suite for RegisterWithPassword ---

type RegisterWithPasswordServiceTestSuite struct {
	suite.Suite
	mockUserPoolRepo   *mock.MockUserPoolRepository
	mockUserRepo       *mock.MockUserRepository
	mockUserService    *mock.MockUserService
	mockProfileService *mock.MockProfileService
	mockHashService    *mock.MockHashService
	mockOtpService     *mock.MockOtpService
	mockSessionService *mock.MockSessionService
	mockAuthService    *mock.MockAuthorizeService

	registerService rs.IRegisterService
}

func (this *RegisterWithPasswordServiceTestSuite) SetupTest() {
	this.mockUserPoolRepo = new(mock.MockUserPoolRepository)
	this.mockUserRepo = new(mock.MockUserRepository)
	this.mockUserService = new(mock.MockUserService)
	this.mockProfileService = new(mock.MockProfileService)
	this.mockHashService = new(mock.MockHashService)
	this.mockOtpService = new(mock.MockOtpService)
	this.mockSessionService = new(mock.MockSessionService)
	this.mockAuthService = new(mock.MockAuthorizeService)
	logger := zap.NewNop()

	this.registerService = rs.NewRegisterService(
		this.mockUserPoolRepo,
		this.mockUserRepo,
		this.mockUserService,
		this.mockProfileService,
		logger,
		this.mockHashService,
		this.mockOtpService,
		this.mockSessionService,
		this.mockAuthService,
	)
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_AppDoesNotAllowPasswordLogin_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_OTP"}, // No password login
		UsersPool:  entity.UsersPool{ID: "pool-id"},
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowNotAllowed(""), err)

	if ge, ok := err.(*e.AppError); ok {
		assert.Equal(this.T(), 405, ge.Code.Second)
		assert.Equal(this.T(), e.AppErrorCode("NOT_ALLOWED"), ge.Code.First)
	}
}

// An app asking for email verification no longer makes the service refuse a
// password registration: that decision moved out to a guard, so every entrypoint
// gets gated the same way instead of this one alone. What the service still owes
// is the state the guard reads afterwards - the user is created unverified, no
// matter what the app asks for - which is what this test pins down.
//
// TODO: cover the refusal itself in the suite of the email verification guard
// once it lands. Until then no test asserts that a verification-required app
// blocks password registration.
func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_AppRequiresEmailVerification_IsLeftToTheGuard() {
	// Arrange
	app := &entity.App{
		ID:          "app-id",
		LoginTypes:  []string{"WITH_PASSWORD"},
		VerifyEmail: true, // Requires email verification
		UsersPool:   entity.UsersPool{ID: "pool-id"},
		Role:        "USER",
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	profile := &entity.Profile{
		ID:   "profile-id",
		Name: "User Profile",
	}
	createdUser := &entity.User{
		ID:           1,
		Email:        "test@example.com",
		Name:         "Test User",
		UsersPoolId:  "pool-id",
		PasswordHash: "hashed_password",
		ProfileId:    "profile-id",
	}
	session := &entity.Session{
		ID:     "session-id",
		UserId: 1,
		AppId:  "app-id",
	}
	credentials := &services.AuthorizationCredentials{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	// Mocks
	this.mockUserService.On("IsAlreadyCreated", "test@example.com", app).Return(false, nil)
	this.mockHashService.On("HashText", "password123", testifymock.Anything).Return("hashed_password", nil)
	this.mockProfileService.On("GetProfileByAppRole", "USER").Return(profile, nil)
	// The matcher is the assertion: an app requiring verification does not verify
	// anything by itself, so the user has to reach the repository unverified. A
	// service that flagged it as verified would not match and the call would fail.
	this.mockUserRepo.On("Create", testifymock.MatchedBy(func(user entity.User) bool {
		return !user.VerifyEmail
	})).Return(createdUser, nil)
	this.mockUserRepo.On("FindWhere", entity.User{ID: uint(1)}, []string{"Profile"}).Return(createdUser, nil)
	this.mockSessionService.On("CreateNew", app, createdUser, requestInfo, "WITH_PASSWORD").Return(session, nil)
	this.mockAuthService.On("CreateAuthorizationCredentials", app, session).Return(credentials, nil)

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), response)
	assert.Equal(this.T(), "session-id", response.SessionId)

	this.mockUserRepo.AssertExpectations(this.T())
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_UserAlreadyExists_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
		Role:       "USER",
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	// Mock user already exists
	this.mockUserService.On("IsAlreadyCreated", "test@example.com", app).Return(true, nil)

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowBadRequest(""), err)
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_Success() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
		Role:       "USER",
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	profile := &entity.Profile{
		ID:   "profile-id",
		Name: "User Profile",
	}
	createdUser := &entity.User{
		ID:           1,
		Email:        "test@example.com",
		Name:         "Test User",
		UsersPoolId:  "pool-id",
		PasswordHash: "hashed_password",
		ProfileId:    "profile-id",
	}
	session := &entity.Session{
		ID:     "session-id",
		UserId: 1,
		AppId:  "app-id",
	}
	credentials := &services.AuthorizationCredentials{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	// Mocks
	this.mockUserService.On("IsAlreadyCreated", "test@example.com", app).Return(false, nil)
	this.mockHashService.On("HashText", "password123", testifymock.Anything).Return("hashed_password", nil)
	this.mockProfileService.On("GetProfileByAppRole", "USER").Return(profile, nil)
	this.mockUserRepo.On("Create", testifymock.Anything).Return(createdUser, nil)
	this.mockUserRepo.On("FindWhere", entity.User{ID: uint(1)}, []string{"Profile"}).Return(createdUser, nil)
	this.mockSessionService.On("CreateNew", app, createdUser, requestInfo, "WITH_PASSWORD").Return(session, nil)
	this.mockAuthService.On("CreateAuthorizationCredentials", app, session).Return(credentials, nil)

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), response)
	assert.Equal(this.T(), "session-id", response.SessionId)
	assert.Equal(this.T(), "access-token", response.AccessToken)
	assert.Equal(this.T(), "refresh-token", response.RefreshToken)
	assert.Equal(this.T(), "test@example.com", response.User.Email)
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_HashPasswordFails_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
		Role:       "USER",
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	// Mocks
	this.mockUserService.On("IsAlreadyCreated", "test@example.com", app).Return(false, nil)
	this.mockHashService.On("HashText", "password123", testifymock.Anything).Return("", assert.AnError)

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowInternalServerError(""), err)
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_ProfileNotFound_Error() {
	// Arrange
	app := &entity.App{
		ID:         "app-id",
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: "pool-id"},
		Role:       "USER",
	}
	metadata, _ := json.Marshal(map[string]string{"test": "value"})
	payload := dto.RegisterPayloadWithPassoword{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
		Metadata: metadata,
	}
	requestInfo := dto.RequestInfo{}

	// Mocks
	this.mockUserService.On("IsAlreadyCreated", "test@example.com", app).Return(false, nil)
	this.mockHashService.On("HashText", "password123", testifymock.Anything).Return("hashed_password", nil)
	this.mockProfileService.On("GetProfileByAppRole", "USER").Return(nil, assert.AnError)

	// Act
	response, err := this.registerService.RegisterWithPassword(app, payload, requestInfo)

	// Assert
	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowInternalServerError(""), err)
}

func TestRegisterServiceSuite(t *testing.T) {
	suite.Run(t, new(RegisterWithPasswordServiceTestSuite))
}
