package user_test

import (
	"encoding/json"
	"testing"
	"time"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/user/models"
	"auth_service/app/modules/core/user/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
	"auth_service/shared/tracking"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	organizationId = "11111111-1111-1111-1111-111111111111"
	poolId         = "22222222-2222-2222-2222-222222222222"
	foreignPoolId  = "33333333-3333-3333-3333-333333333333"
	targetUserId   = uint(7)
)

type UserUpdateTestSuite struct {
	suite.Suite
	userRepository  *mock.MockUserRepository
	appRepository   *mock.MockAppRepository
	userPoolService *mock.MockUserPoolService
	txManager       *mock.MockTransactionManager
	service         *services.UserService

	organization *entity.Organization
}

func (this *UserUpdateTestSuite) SetupTest() {
	this.userRepository = new(mock.MockUserRepository)
	this.appRepository = new(mock.MockAppRepository)
	this.userPoolService = new(mock.MockUserPoolService)
	this.txManager = new(mock.MockTransactionManager)
	this.service = services.NewUserService(
		this.userRepository,
		this.appRepository,
		this.userPoolService,
		this.txManager,
		zap.NewNop(),
	)
	this.organization = &entity.Organization{ID: organizationId}
}

func (this *UserUpdateTestSuite) storedUser(metadata string) *entity.User {
	return &entity.User{
		ID:          targetUserId,
		Email:       "target@example.com",
		Name:        "Target",
		UsersPoolId: poolId,
		Metadata:    json.RawMessage(metadata),
	}
}

// The pool of the target belongs to the current organization, which is the whole
// scope rule of the administration route.
func (this *UserUpdateTestSuite) expectOwnedPool() {
	this.userPoolService.
		On("FindByIdInOrganization", poolId, organizationId).
		Return(&entity.UsersPool{ID: poolId}, nil)
}

func (this *UserUpdateTestSuite) expectFindOne(user *entity.User) {
	this.userRepository.
		On("FindOne", entity.User{ID: user.ID}, testifymock.Anything).
		Return(user, nil)
}

func codeOf(t assert.TestingT, err error) e.AppErrorCode {
	appError, ok := err.(*e.AppError)

	if !assert.True(t, ok, "expected an AppError, got %v", err) {
		return ""
	}

	return appError.Code.First
}

func pointer[T any](value T) *T {
	return &value
}

func (this *UserUpdateTestSuite) TestMetadataIsMerged() {
	stored := this.storedUser(`{"kept": 1, "gone": 2}`)

	this.expectFindOne(stored)
	this.expectOwnedPool()

	this.userRepository.
		On("Update", entity.User{ID: targetUserId}, testifymock.MatchedBy(func(dao dto.UserUpdateDao) bool {
			if dao.Metadata == nil {
				return false
			}

			merged := map[string]any{}
			_ = json.Unmarshal(*dao.Metadata, &merged)

			_, removed := merged["gone"]

			return merged["kept"] == float64(1) && merged["added"] == "yes" && !removed
		}), testifymock.Anything).
		Return(int64(1), nil)

	patch := json.RawMessage(`{"added": "yes", "gone": null}`)

	_, err := this.service.UpdateForOrganization(this.organization, targetUserId, &dto.UpdateUser{Metadata: &patch})

	assert.NoError(this.T(), err)
	this.userRepository.AssertExpectations(this.T())
}

// The administration route has no self escape: FindById lets a user read itself
// before checking the pool, and inheriting that branch here would let anybody
// edit itself through /core/users/{id} whatever organization it is scoped to.
// Editing yourself is PUT /core/users/me, which is a route with its own grant.
func (this *UserUpdateTestSuite) TestNoSelfEscapeOnTheAdministrationRoute() {
	caller := this.storedUser(`{}`)
	caller.UsersPoolId = foreignPoolId

	this.expectFindOne(caller)

	this.userPoolService.
		On("FindByIdInOrganization", foreignPoolId, organizationId).
		Return(nil, nil)

	updated, err := this.service.UpdateForOrganization(this.organization, caller.ID, &dto.UpdateUser{
		Name: pointer("Renamed"),
	})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("NOT_FOUND"), codeOf(this.T(), err))
	this.userRepository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// The self route needs no pool lookup at all: the target is the caller, resolved
// by the AuthGuard and never read from the request.
func (this *UserUpdateTestSuite) TestTheSelfRouteChecksNoPool() {
	caller := this.storedUser(`{}`)
	caller.UsersPoolId = foreignPoolId

	this.userRepository.
		On("Update", entity.User{ID: caller.ID}, testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)
	this.expectFindOne(caller)

	_, err := this.service.UpdateSelf(caller, &dto.UpdateUserSelf{Name: pointer("Renamed")})

	assert.NoError(this.T(), err)
	this.userPoolService.AssertNotCalled(this.T(), "FindByIdInOrganization", testifymock.Anything, testifymock.Anything)
}

// A collision on the unique index would surface as a 500 from the database, so
// the service names the conflict instead.
func (this *UserUpdateTestSuite) TestATakenEmailIsAConflict() {
	stored := this.storedUser(`{}`)

	this.expectFindOne(stored)
	this.expectOwnedPool()

	this.userRepository.
		On("FindOne", entity.User{Email: "taken@example.com", UsersPoolId: poolId}, testifymock.Anything).
		Return(&entity.User{ID: 99}, nil)

	updated, err := this.service.UpdateForOrganization(this.organization, targetUserId, &dto.UpdateUser{
		Email: pointer("Taken@Example.com"),
	})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("USER_ALREADY_EXISTS"), codeOf(this.T(), err))
}

// Emails are stored lowercased everywhere else, so an update has to normalise too
// or the unique index stops catching duplicates that differ only in case.
func (this *UserUpdateTestSuite) TestAnEmailIsLowercasedBeforeBeingWritten() {
	stored := this.storedUser(`{}`)

	this.expectFindOne(stored)
	this.expectOwnedPool()

	this.userRepository.
		On("FindOne", entity.User{Email: "new@example.com", UsersPoolId: poolId}, testifymock.Anything).
		Return(nil, nil)

	this.userRepository.
		On("Update", entity.User{ID: targetUserId}, testifymock.MatchedBy(func(dao dto.UserUpdateDao) bool {
			return dao.Email != nil && *dao.Email == "new@example.com"
		}), testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.UpdateForOrganization(this.organization, targetUserId, &dto.UpdateUser{
		Email: pointer("NEW@Example.com"),
	})

	assert.NoError(this.T(), err)
	this.userRepository.AssertExpectations(this.T())
}

func (this *UserUpdateTestSuite) TestAnEmptyPayloadTouchesNothing() {
	stored := this.storedUser(`{}`)

	this.expectFindOne(stored)
	this.expectOwnedPool()

	updated, err := this.service.UpdateForOrganization(this.organization, targetUserId, &dto.UpdateUser{})

	assert.NoError(this.T(), err)
	assert.Equal(this.T(), stored.ID, updated.ID)
	this.userRepository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func TestUserUpdateSuite(t *testing.T) {
	suite.Run(t, new(UserUpdateTestSuite))
}

// --- Track ---

func (this *UserUpdateTestSuite) TestTrackPushesTheLoginNewestFirst() {
	tx := &repo.Tx{}
	option := []repo.Option{{Tx: tx}}

	stored := this.storedUser(`{}`)
	stored.Tracking = json.RawMessage(`{"login":{"events":[{"app_id":"older"}]}}`)

	this.txManager.On("Tx").Return(tx, nil)
	this.userRepository.On("FindOneForUpdate", targetUserId, option).Return(stored, nil)

	this.userRepository.
		On("WriteTracking", targetUserId, testifymock.MatchedBy(func(document json.RawMessage) bool {
			parsed, err := tracking.Parse(document)

			return err == nil &&
				parsed.Login != nil &&
				len(parsed.Login.Events) == 2 &&
				parsed.Login.Events[0].AppId == "newer" &&
				parsed.Login.Events[0].Ip == "203.0.113.7"
		}), option).
		Return(int64(1), nil)

	err := this.service.Track(tracking.TagLogin, services.TrackLogin{
		UserId: targetUserId,
		Event:  tracking.LoginEvent{AppId: "newer", Ip: "203.0.113.7", At: time.Now()},
	})

	assert.NoError(this.T(), err)
	this.userRepository.AssertExpectations(this.T())
	this.userRepository.AssertCalled(this.T(), "FindOneForUpdate", targetUserId, option)
}

// A user does not track signups. Refused rather than ignored, so a wrong wiring is
// loud instead of silently dropping every event.
func (this *UserUpdateTestSuite) TestTrackRefusesATagTheUserDoesNotKeep() {
	err := this.service.Track(tracking.TagSignup, services.TrackLogin{UserId: targetUserId})

	assert.Error(this.T(), err)
	this.txManager.AssertNotCalled(this.T(), "Tx")
}

func (this *UserUpdateTestSuite) TestTrackNeedsTheUser() {
	assert.Error(this.T(), this.service.Track(tracking.TagLogin, services.TrackLogin{}))

	this.txManager.AssertNotCalled(this.T(), "Tx")
}

// The session that was just minted is the login event, so nothing has to be
// gathered twice - and reading it into a value keeps the goroutine off a struct
// the caller goes on to mutate.
func (this *UserUpdateTestSuite) TestLoginFromReadsTheSession() {
	moment := time.Now()

	payload := services.LoginFrom(&entity.Session{
		UserId:    targetUserId,
		AppId:     "app-1",
		IpAddress: "203.0.113.7",
		UserAgent: "curl/8.0",
		LoginType: "WITH_OTP",
		CreatedAt: moment,
	})

	assert.Equal(this.T(), targetUserId, payload.UserId)
	assert.Equal(this.T(), "app-1", payload.Event.AppId)
	assert.Equal(this.T(), "203.0.113.7", payload.Event.Ip)
	assert.Equal(this.T(), "curl/8.0", payload.Event.UserAgent)
	assert.Equal(this.T(), "WITH_OTP", payload.Event.LoginType)
	assert.True(this.T(), moment.Equal(payload.Event.At))
}

// The tracking column is not in the update dao, which is what makes it
// unreachable from PUT /core/users/{id} - see docs/steering/models-layer.md.
func (this *UserUpdateTestSuite) TestTheUpdateDaoCannotWriteTracking() {
	document := json.RawMessage(`{"login":{}}`)

	written := repo.BuildUpdateMap(dto.UserUpdateDao{
		Name:     pointer("Renamed"),
		Metadata: &document,
	})

	assert.Contains(this.T(), written, "Name")
	assert.NotContains(this.T(), written, "Tracking")
}

// entity.User is embedded in the login, register, refresh and listing responses,
// so the column has to be invisible through the entity and come back only from the
// response of the two individual reads.
func (this *UserUpdateTestSuite) TestTheEntityNeverSerializesTracking() {
	user := this.storedUser(`{}`)
	user.Tracking = json.RawMessage(`{"login":{"events":[{"ip":"203.0.113.7"}]}}`)

	encoded, err := json.Marshal(user)

	assert.NoError(this.T(), err)
	assert.NotContains(this.T(), string(encoded), "tracking")
	assert.NotContains(this.T(), string(encoded), "203.0.113.7")

	exposed, err := json.Marshal(dto.UserWithTracking(user))

	assert.NoError(this.T(), err)
	assert.Contains(this.T(), string(exposed), `"tracking"`)
	assert.Contains(this.T(), string(exposed), "203.0.113.7")
}
