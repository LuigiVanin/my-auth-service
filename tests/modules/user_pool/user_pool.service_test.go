package user_pool_test

import (
	"encoding/json"
	"testing"
	"time"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/user_pool/models"
	"auth_service/app/modules/core/user_pool/services"
	entity "auth_service/infra/entities"
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
	organizationId = "11111111-1111-1111-1111-111111111111"
	otherOrgId     = "22222222-2222-2222-2222-222222222222"
	poolId         = "33333333-3333-3333-3333-333333333333"
)

type UserPoolUpdateTestSuite struct {
	suite.Suite
	repository     *mock.MockUserPoolRepository
	profileService *mock.MockProfileService
	cipherService  *mock.MockCipherService
	txManager      *mock.MockTransactionManager
	service        *services.UserPoolService
}

func (this *UserPoolUpdateTestSuite) SetupTest() {
	this.repository = new(mock.MockUserPoolRepository)
	this.profileService = new(mock.MockProfileService)
	this.cipherService = new(mock.MockCipherService)
	this.txManager = new(mock.MockTransactionManager)
	this.service = services.NewUserPoolService(
		this.repository,
		this.profileService,
		this.cipherService,
		this.txManager,
		zap.NewNop(),
	)
}

// The caller and its organization both hold the same document, which is the
// ceiling a requested default profile is compared against.
func callerHolding(grants ...string) (*entity.Organization, *entity.Participant) {
	document, _ := json.Marshal(permissions.Document{Grants: grants})

	return &entity.Organization{
			ID:      organizationId,
			Profile: &entity.Profile{Permissions: document},
		},
		&entity.Participant{
			ID:      "participant-id",
			Profile: &entity.Profile{Permissions: document},
		}
}

func profileGranting(id string, grants ...string) *entity.Profile {
	document, _ := json.Marshal(permissions.Document{Grants: grants})

	return &entity.Profile{ID: id, Key: "CANDIDATE", Permissions: document}
}

func (this *UserPoolUpdateTestSuite) expectFindOne(pool *entity.UsersPool) {
	this.repository.
		On("FindOne", entity.UsersPool{ID: poolId}, testifymock.Anything).
		Return(pool, nil)
}

func (this *UserPoolUpdateTestSuite) storedPool(metadata string) *entity.UsersPool {
	scope := organizationId

	return &entity.UsersPool{
		ID:               poolId,
		Name:             "Clients",
		OrganizationId:   &scope,
		DefaultProfileId: "current-profile-id",
		Metadata:         json.RawMessage(metadata),
	}
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

func (this *UserPoolUpdateTestSuite) TestNameAndDescriptionAreWritten() {
	organization, participant := callerHolding("as::users::READ")

	this.expectFindOne(this.storedPool(`{}`))

	this.repository.
		On("Update", entity.UsersPool{ID: poolId}, testifymock.MatchedBy(func(dao dto.UserPoolUpdateDao) bool {
			return dao.Name != nil && *dao.Name == "Renamed" &&
				dao.Description != nil && *dao.Description == "" &&
				dao.DefaultProfileId == nil
		}), testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{
		Name:        pointer("Renamed"),
		Description: pointer(""),
	})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

func (this *UserPoolUpdateTestSuite) TestMetadataIsMerged() {
	organization, participant := callerHolding("as::users::READ")

	this.expectFindOne(this.storedPool(`{"kept": 1, "gone": 2}`))

	this.repository.
		On("Update", entity.UsersPool{ID: poolId}, testifymock.MatchedBy(func(dao dto.UserPoolUpdateDao) bool {
			if dao.Metadata == nil {
				return false
			}

			merged := map[string]any{}
			_ = json.Unmarshal(*dao.Metadata, &merged)

			_, removed := merged["gone"]

			return merged["kept"] == float64(1) && merged["added"] == true && !removed
		}), testifymock.Anything).
		Return(int64(1), nil)

	patch := json.RawMessage(`{"added": true, "gone": null}`)

	_, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{Metadata: &patch})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

// The default profile of a pool is the ceiling every organization born in it
// receives, so moving it is the one field of this payload that can hand out more
// than the caller holds. It goes through the same clamp as creation.
func (this *UserPoolUpdateTestSuite) TestANewDefaultProfileWiderThanTheCallerIsRefused() {
	organization, participant := callerHolding("as::users::READ")

	this.expectFindOne(this.storedPool(`{}`))

	this.profileService.
		On("FindByIdVisibleTo", "wider-profile-id", organizationId).
		Return(profileGranting("wider-profile-id", "as::users::READ", "as::apps::CREATE"), nil)

	updated, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{
		DefaultProfileId: pointer("wider-profile-id"),
	})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("PERMISSION_DENIED"), codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func (this *UserPoolUpdateTestSuite) TestADefaultProfileInsideTheCeilingIsWritten() {
	organization, participant := callerHolding("as::users::READ", "as::apps::READ")

	this.expectFindOne(this.storedPool(`{}`))

	this.profileService.
		On("FindByIdVisibleTo", "narrow-profile-id", organizationId).
		Return(profileGranting("narrow-profile-id", "as::users::READ"), nil)

	this.repository.
		On("Update", entity.UsersPool{ID: poolId}, testifymock.MatchedBy(func(dao dto.UserPoolUpdateDao) bool {
			return dao.DefaultProfileId != nil && *dao.DefaultProfileId == "narrow-profile-id"
		}), testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{
		DefaultProfileId: pointer("narrow-profile-id"),
	})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

// Folded into "does not exist", never 403, so the id cannot be used to probe for
// the pools of other organizations.
func (this *UserPoolUpdateTestSuite) TestAPoolOfAnotherOrganizationIsNotFound() {
	organization, participant := callerHolding("as::users::READ")

	other := otherOrgId
	stored := this.storedPool(`{}`)
	stored.OrganizationId = &other

	this.expectFindOne(stored)

	updated, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{
		Name: pointer("Renamed"),
	})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("NOT_FOUND"), codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func (this *UserPoolUpdateTestSuite) TestAnEmptyPayloadTouchesNothing() {
	organization, participant := callerHolding("as::users::READ")

	stored := this.storedPool(`{}`)
	this.expectFindOne(stored)

	updated, err := this.service.Update(poolId, organization, participant, &dto.UpdateUserPool{})

	assert.NoError(this.T(), err)
	assert.Equal(this.T(), stored.ID, updated.ID)
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// The create payload used to accept a description and drop it on the floor: the
// column did not exist and CreateUserPoolData had no field for it.
func (this *UserPoolUpdateTestSuite) TestCreateWritesTheDescription() {
	organization, participant := callerHolding("as::users::READ")

	this.profileService.
		On("FindByIdVisibleTo", "narrow-profile-id", organizationId).
		Return(profileGranting("narrow-profile-id", "as::users::READ"), nil)

	this.repository.
		On("Create", testifymock.MatchedBy(func(pool entity.UsersPool) bool {
			return pool.Description == "The clients of the tenant"
		}), testifymock.Anything).
		Return(&entity.UsersPool{ID: poolId}, nil)

	this.cipherService.
		On("EncryptUuidIntoToken", poolId, testifymock.Anything).
		Return("as_public_key", nil)

	this.repository.
		On("Update", entity.UsersPool{ID: poolId}, testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.Create(services.CreateUserPoolData{
		Name:             "Clients",
		Description:      "The clients of the tenant",
		OrganizationId:   &organization.ID,
		DefaultProfileId: "narrow-profile-id",
	}, organization, participant)

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

func TestUserPoolUpdateSuite(t *testing.T) {
	suite.Run(t, new(UserPoolUpdateTestSuite))
}

// --- Track ---

func (this *UserPoolUpdateTestSuite) expectTrackTransaction(stored *entity.UsersPool) *repo.Tx {
	tx := &repo.Tx{}
	option := []repo.Option{{Tx: tx}}

	this.txManager.On("Tx").Return(tx, nil)
	this.repository.On("FindOneForUpdate", poolId, option).Return(stored, nil)

	return tx
}

func (this *UserPoolUpdateTestSuite) TestTrackCountsTheMonthAndTheApp() {
	stored := this.storedPool(`{}`)
	stored.Tracking = json.RawMessage(`{}`)

	tx := this.expectTrackTransaction(stored)

	this.repository.
		On("WriteTracking", poolId, testifymock.MatchedBy(func(document json.RawMessage) bool {
			parsed, err := tracking.Parse(document)

			return err == nil &&
				parsed.Signup != nil &&
				parsed.Signup.Periods["2026-09"].Total == 1 &&
				parsed.Signup.Periods["2026-09"].Apps["app-1"] == 1
		}), []repo.Option{{Tx: tx}}).
		Return(int64(1), nil)

	err := this.service.Track(tracking.TagSignup, services.TrackSignup{
		PoolId: poolId,
		AppId:  "app-1",
		At:     time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
	})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

// The read is the locked one: without it two concurrent signups would each write
// the document they read and one increment would vanish.
func (this *UserPoolUpdateTestSuite) TestTrackReadsUnderALock() {
	stored := this.storedPool(`{}`)
	stored.Tracking = json.RawMessage(`{}`)

	tx := this.expectTrackTransaction(stored)

	this.repository.
		On("WriteTracking", poolId, testifymock.Anything, []repo.Option{{Tx: tx}}).
		Return(int64(1), nil)

	assert.NoError(this.T(), this.service.Track(tracking.TagSignup, services.TrackSignup{
		PoolId: poolId,
		AppId:  "app-1",
		At:     time.Now(),
	}))

	this.repository.AssertCalled(this.T(), "FindOneForUpdate", poolId, []repo.Option{{Tx: tx}})
	this.repository.AssertNotCalled(this.T(), "FindOne", testifymock.Anything, testifymock.Anything)
}

// A pool does not track logins. Refused rather than ignored, so a wrong wiring is
// loud instead of silently dropping every event.
func (this *UserPoolUpdateTestSuite) TestTrackRefusesATagThePoolDoesNotKeep() {
	err := this.service.Track(tracking.TagLogin, services.TrackSignup{PoolId: poolId, AppId: "app-1"})

	assert.Error(this.T(), err)
	this.txManager.AssertNotCalled(this.T(), "Tx")
}

func (this *UserPoolUpdateTestSuite) TestTrackNeedsThePoolAndTheApp() {
	assert.Error(this.T(), this.service.Track(tracking.TagSignup, services.TrackSignup{AppId: "app-1"}))
	assert.Error(this.T(), this.service.Track(tracking.TagSignup, services.TrackSignup{PoolId: poolId}))

	this.txManager.AssertNotCalled(this.T(), "Tx")
}

// The tracking column is not in the update dao, which is what makes it
// unreachable from PUT /core/users_pool/{id} - see steering/models-layer.md.
func (this *UserPoolUpdateTestSuite) TestTheUpdateDaoCannotWriteTracking() {
	document := json.RawMessage(`{"signup":{}}`)

	written := repo.BuildUpdateMap(dto.UserPoolUpdateDao{
		Name:     pointer("Renamed"),
		Metadata: &document,
	})

	assert.Contains(this.T(), written, "Name")
	assert.NotContains(this.T(), written, "Tracking")
	assert.NotContains(this.T(), written, "UsersCount")
}
