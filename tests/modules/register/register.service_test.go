package register_test

import (
	"encoding/json"
	"testing"
	"time"

	e "auth_service/app/errors"
	"auth_service/app/modules/authorize/services"
	odto "auth_service/app/modules/core/organization/models"
	ps "auth_service/app/modules/core/participant/services"
	us "auth_service/app/modules/core/user/services"
	ups "auth_service/app/modules/core/user_pool/services"
	dto "auth_service/app/modules/register/models"
	rs "auth_service/app/modules/register/services"
	entity "auth_service/infra/entities"
	sharedDto "auth_service/shared/models"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"
	"auth_service/shared/tracking"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	poolId       = "pool-id"
	appId        = "app-id"
	profileId    = "profile-id"
	adminId      = "admin-profile-id"
	orgId        = "organization-id"
	userEmail    = "test@example.com"
	userPassword = "password123"
)

// The four writes of a provisioning go straight to the repositories, so the suite
// mocks those and not the services that wrap them - see
// docs/steering/service-layer.md.
type RegisterWithPasswordServiceTestSuite struct {
	suite.Suite
	mockUserRepo         *mock.MockUserRepository
	mockOrganizationRepo *mock.MockOrganizationRepository
	mockParticipantRepo  *mock.MockParticipantRepository
	mockProfileRepo      *mock.MockProfileRepository
	mockUserPoolRepo     *mock.MockUserPoolRepository
	mockUserPoolService  *mock.MockUserPoolService
	mockUserService      *mock.MockUserService
	mockParticipantSvc   *mock.MockParticipantService
	mockHashService      *mock.MockHashService
	mockOtpService       *mock.MockOtpService
	mockSessionService   *mock.MockSessionService
	mockAuthService      *mock.MockAuthorizeService
	mockTxManager        *mock.MockTransactionManager

	tx     *repo.Tx
	option []repo.Option

	registerService rs.IRegisterService
}

func (this *RegisterWithPasswordServiceTestSuite) SetupTest() {
	this.mockUserRepo = new(mock.MockUserRepository)
	this.mockOrganizationRepo = new(mock.MockOrganizationRepository)
	this.mockParticipantRepo = new(mock.MockParticipantRepository)
	this.mockProfileRepo = new(mock.MockProfileRepository)
	this.mockUserPoolRepo = new(mock.MockUserPoolRepository)
	this.mockUserPoolService = new(mock.MockUserPoolService)
	this.mockUserService = new(mock.MockUserService)
	this.mockParticipantSvc = new(mock.MockParticipantService)
	this.mockHashService = new(mock.MockHashService)
	this.mockOtpService = new(mock.MockOtpService)
	this.mockSessionService = new(mock.MockSessionService)
	this.mockAuthService = new(mock.MockAuthorizeService)
	this.mockTxManager = new(mock.MockTransactionManager)

	// A zero Tx carries no client, so Commit and Rollback are no ops and every
	// repository call inside the unit of work is still matched on the same option.
	this.tx = &repo.Tx{}
	this.option = []repo.Option{{Tx: this.tx}}

	this.registerService = rs.NewRegisterService(
		this.mockUserRepo,
		this.mockOrganizationRepo,
		this.mockParticipantRepo,
		this.mockProfileRepo,
		this.mockUserPoolRepo,
		this.mockUserPoolService,
		this.mockUserService,
		this.mockParticipantSvc,
		zap.NewNop(),
		this.mockHashService,
		this.mockOtpService,
		this.mockSessionService,
		this.mockAuthService,
		this.mockTxManager,
	)
}

func passwordApp() *entity.App {
	return &entity.App{
		ID:         appId,
		LoginTypes: []string{"WITH_PASSWORD"},
		UsersPool:  entity.UsersPool{ID: poolId, DefaultProfileId: profileId},
	}
}

func passwordPayload() dto.RegisterPayloadWithPassoword {
	metadata, _ := json.Marshal(map[string]string{"test": "value"})

	return dto.RegisterPayloadWithPassoword{
		Email:    userEmail,
		Password: userPassword,
		Name:     "Test User",
		Metadata: metadata,
	}
}

// Everything a successful provisioning writes, in the order the cycle between the
// tables forces: organization with no owner, its Admin profile, the user, the
// owner stamped, the participation, then the counter of the pool.
func (this *RegisterWithPasswordServiceTestSuite) arrangeProvisioning(
	createdUser *entity.User,
) (chan ups.TrackSignup, chan us.TrackLogin) {
	organization := &entity.Organization{ID: orgId, UsersPoolId: poolId, ProfileId: profileId}
	admin := &entity.Profile{ID: adminId}
	participation := &entity.Participant{ID: "participant-id", OrganizationId: orgId, Profile: admin}

	this.mockTxManager.On("Tx").Return(this.tx, nil)

	this.mockOrganizationRepo.
		On("Create", testifymock.Anything, this.option).
		Return(organization, nil)

	this.mockProfileRepo.
		On("Create", testifymock.Anything, this.option).
		Return(admin, nil)

	this.mockUserRepo.
		On("Create", testifymock.Anything, this.option).
		Return(createdUser, nil)

	this.mockOrganizationRepo.
		On("Update", entity.Organization{ID: orgId}, odto.OrganizationUpdateDao{OwnerUserId: &createdUser.ID}, this.option).
		Return(int64(1), nil)

	this.mockParticipantRepo.
		On("Create", testifymock.Anything, this.option).
		Return(participation, nil)

	this.mockUserPoolRepo.
		On("IncrementUsersCount", poolId, this.option).
		Return(int64(1), nil)

	this.mockUserRepo.
		On("FindOne", entity.User{ID: createdUser.ID}, []repo.Option{{
			With: []string{"CurrentOrganization", "CurrentOrganization.Profile"},
		}}).
		Return(createdUser, nil)

	this.mockParticipantSvc.
		On("FindForCurrentOrganization", createdUser).
		Return(&ps.ResolvedParticipation{
			Participant: participation,
			Permissions: &permissions.Resolved{},
		}, nil)

	// Both tracks are detached, and Detach recovers from a panic - so an unstubbed
	// call here would be swallowed instead of failing the test. The channels are
	// what make the assertion deterministic.
	//
	// They are locals and not fields on the suite: a detached goroutine outlives the
	// test that started it, so a field would be read by the previous test's
	// goroutine while this one reassigns it.
	trackedSignup := make(chan ups.TrackSignup, 1)
	trackedLogin := make(chan us.TrackLogin, 1)

	this.mockUserPoolService.
		On("Track", tracking.TagSignup, testifymock.Anything).
		Run(func(args testifymock.Arguments) {
			trackedSignup <- args.Get(1).(ups.TrackSignup)
		}).
		Return(nil)

	this.mockUserService.
		On("Track", tracking.TagLogin, testifymock.Anything).
		Run(func(args testifymock.Arguments) {
			trackedLogin <- args.Get(1).(us.TrackLogin)
		}).
		Return(nil)

	return trackedSignup, trackedLogin
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_AppDoesNotAllowPasswordLogin_Error() {
	app := passwordApp()
	app.LoginTypes = []string{"WITH_OTP"}

	response, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)

	if appError, ok := err.(*e.AppError); ok {
		assert.Equal(this.T(), 405, appError.Code.Second)
		assert.Equal(this.T(), e.AppErrorCode("NOT_ALLOWED"), appError.Code.First)
	}
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_UserAlreadyExists_Error() {
	app := passwordApp()

	this.mockUserService.On("IsAlreadyCreated", userEmail, app).Return(true, nil)

	response, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_HashPasswordFails_Error() {
	app := passwordApp()

	this.mockUserService.On("IsAlreadyCreated", userEmail, app).Return(false, nil)
	this.mockHashService.On("HashText", userPassword, testifymock.Anything).Return("", assert.AnError)

	response, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.Nil(this.T(), response)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowInternalServerError(""), err)
}

// An app asking for email verification no longer makes the service refuse a
// password registration: that decision moved out to a guard, so every entrypoint
// gets gated the same way instead of this one alone. What the service still owes
// is the state the guard reads afterwards - the user is created unverified, no
// matter what the app asks for - which is what the matcher below pins down.
//
// TODO: cover the refusal itself in the suite of the email verification guard
// once it lands. Until then no test asserts that a verification-required app
// blocks password registration.
func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_AppRequiresEmailVerification_IsLeftToTheGuard() {
	app := passwordApp()
	app.VerifyEmail = true

	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	this.mockUserService.On("IsAlreadyCreated", userEmail, app).Return(false, nil)
	this.mockHashService.On("HashText", userPassword, testifymock.Anything).Return("hashed_password", nil)
	this.arrangeProvisioning(createdUser)

	// The matcher is the assertion: an app requiring verification does not verify
	// anything by itself, so the user has to reach the repository unverified.
	this.mockUserRepo.ExpectedCalls = nil
	this.mockUserRepo.
		On("Create", testifymock.MatchedBy(func(user entity.User) bool {
			return !user.VerifyEmail
		}), this.option).
		Return(createdUser, nil)
	this.mockUserRepo.
		On("FindOne", entity.User{ID: createdUser.ID}, []repo.Option{{
			With: []string{"CurrentOrganization", "CurrentOrganization.Profile"},
		}}).
		Return(createdUser, nil)

	this.mockSessionService.
		On("CreateNew", app, createdUser, sharedDto.RequestInfo{}, "WITH_PASSWORD").
		Return(&entity.Session{ID: "session-id"}, nil)
	this.mockAuthService.
		On("CreateAuthorizationCredentials", app, testifymock.Anything).
		Return(&services.AuthorizationCredentials{AccessToken: "access-token"}, nil)

	response, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), response)

	this.mockUserRepo.AssertExpectations(this.T())
}

func (this *RegisterWithPasswordServiceTestSuite) TestRegisterWithPassword_Success() {
	app := passwordApp()
	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	this.mockUserService.On("IsAlreadyCreated", userEmail, app).Return(false, nil)
	this.mockHashService.On("HashText", userPassword, testifymock.Anything).Return("hashed_password", nil)
	this.arrangeProvisioning(createdUser)

	this.mockSessionService.
		On("CreateNew", app, createdUser, sharedDto.RequestInfo{}, "WITH_PASSWORD").
		Return(&entity.Session{ID: "session-id"}, nil)
	this.mockAuthService.
		On("CreateAuthorizationCredentials", app, testifymock.Anything).
		Return(&services.AuthorizationCredentials{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}, nil)

	response, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), response)
	assert.Equal(this.T(), "session-id", response.SessionId)
	assert.Equal(this.T(), "access-token", response.AccessToken)
	assert.Equal(this.T(), "refresh-token", response.RefreshToken)
	assert.Equal(this.T(), userEmail, response.User.Email)
}

// The counter of the pool is part of the unit of work, so it carries the same
// option as every other write - on the pool client it would survive a rollback.
func (this *RegisterWithPasswordServiceTestSuite) TestProvisionUser_IncrementsTheUsersCountOfThePoolInTheTransaction() {
	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	this.arrangeProvisioning(createdUser)

	provisioned, err := this.registerService.ProvisionUser(passwordApp(), entity.User{Email: userEmail})

	assert.NoError(this.T(), err)
	assert.NotNil(this.T(), provisioned)

	this.mockUserPoolRepo.AssertCalled(this.T(), "IncrementUsersCount", poolId, this.option)
}

func (this *RegisterWithPasswordServiceTestSuite) TestProvisionUser_FailedIncrementRollsTheProvisioningBack() {
	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	this.arrangeProvisioning(createdUser)

	this.mockUserPoolRepo.ExpectedCalls = nil
	this.mockUserPoolRepo.
		On("IncrementUsersCount", poolId, this.option).
		Return(int64(0), assert.AnError)

	provisioned, err := this.registerService.ProvisionUser(passwordApp(), entity.User{Email: userEmail})

	assert.Nil(this.T(), provisioned)
	assert.Error(this.T(), err)
	assert.IsType(this.T(), e.ThrowInternalServerError(""), err)
}

// The signup counter of the pool is tracked, and it is tracked with the app the
// signup came through - which is the whole point of the per-app breakdown.
func (this *RegisterWithPasswordServiceTestSuite) TestProvisionUserTracksTheSignupOfThePool() {
	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	trackedSignup, _ := this.arrangeProvisioning(createdUser)

	_, err := this.registerService.ProvisionUser(passwordApp(), entity.User{Email: userEmail})

	assert.NoError(this.T(), err)

	select {
	case payload := <-trackedSignup:
		assert.Equal(this.T(), poolId, payload.PoolId)
		assert.Equal(this.T(), appId, payload.AppId)
		assert.False(this.T(), payload.At.IsZero())
	case <-time.After(2 * time.Second):
		this.T().Fatal("the signup was never tracked")
	}
}

// A registration is also the first login of the user, so both tracks fire.
func (this *RegisterWithPasswordServiceTestSuite) TestRegisterTracksTheLoginToo() {
	app := passwordApp()
	createdUser := &entity.User{ID: 1, Email: userEmail, UsersPoolId: poolId, CurrentOrganizationId: orgId}

	this.mockUserService.On("IsAlreadyCreated", userEmail, app).Return(false, nil)
	this.mockHashService.On("HashText", userPassword, testifymock.Anything).Return("hashed_password", nil)
	_, trackedLogin := this.arrangeProvisioning(createdUser)

	this.mockSessionService.
		On("CreateNew", app, createdUser, sharedDto.RequestInfo{}, "WITH_PASSWORD").
		Return(&entity.Session{ID: "session-id", UserId: 1, AppId: appId, IpAddress: "203.0.113.7"}, nil)
	this.mockAuthService.
		On("CreateAuthorizationCredentials", app, testifymock.Anything).
		Return(&services.AuthorizationCredentials{AccessToken: "access-token"}, nil)

	_, err := this.registerService.RegisterWithPassword(app, passwordPayload(), sharedDto.RequestInfo{})

	assert.NoError(this.T(), err)

	select {
	case payload := <-trackedLogin:
		assert.Equal(this.T(), uint(1), payload.UserId)
		assert.Equal(this.T(), appId, payload.Event.AppId)
		assert.Equal(this.T(), "203.0.113.7", payload.Event.Ip)
	case <-time.After(2 * time.Second):
		this.T().Fatal("the login was never tracked")
	}
}

func TestRegisterServiceSuite(t *testing.T) {
	suite.Run(t, new(RegisterWithPasswordServiceTestSuite))
}
