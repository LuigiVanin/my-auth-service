package app_test

import (
	"encoding/json"
	"testing"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/app/models"
	"auth_service/app/modules/core/app/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	organizationId = "11111111-1111-1111-1111-111111111111"
	otherOrgId     = "22222222-2222-2222-2222-222222222222"
	appId          = "33333333-3333-3333-3333-333333333333"
)

type AppUpdateTestSuite struct {
	suite.Suite
	repository      *mock.MockAppRepository
	userPoolService *mock.MockUserPoolService
	cipherService   *mock.MockCipherService
	service         *services.AppService

	organization *entity.Organization
}

func (this *AppUpdateTestSuite) SetupTest() {
	this.repository = new(mock.MockAppRepository)
	this.userPoolService = new(mock.MockUserPoolService)
	this.cipherService = new(mock.MockCipherService)
	this.service = services.NewAppService(this.repository, this.userPoolService, this.cipherService)
	this.organization = &entity.Organization{ID: organizationId}
}

func (this *AppUpdateTestSuite) storedApp(metadata string) *entity.App {
	return &entity.App{
		ID:                  appId,
		OrganizationId:      &this.organization.ID,
		Name:                "Portal",
		TokenExpirationTime: 3600,
		Metadata:            json.RawMessage(metadata),
	}
}

func (this *AppUpdateTestSuite) expectFindOne(app *entity.App) {
	this.repository.
		On("FindOne", entity.App{ID: appId}, []repo.Option{{With: []string{"UsersPool"}}}).
		Return(app, nil)
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

// The dao that reaches the repository is the assertion: only the fields the
// payload filled may be in it, or an absent field would overwrite a column.
func (this *AppUpdateTestSuite) TestOnlyTheFieldsSentAreWritten() {
	this.expectFindOne(this.storedApp(`{}`))

	this.repository.
		On("Update", entity.App{ID: appId}, testifymock.MatchedBy(func(dao dto.AppUpdateDao) bool {
			return dao.Name != nil && *dao.Name == "Renamed" &&
				dao.TokenType == nil &&
				dao.TokenExpirationTime == nil &&
				dao.Private == nil &&
				dao.Metadata == nil
		}), testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.Update(appId, this.organization, &dto.UpdateApp{Name: pointer("Renamed")})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

// The reason every field is a pointer: false and 0 are values a caller may want
// written, and the dao has to carry them through.
func (this *AppUpdateTestSuite) TestZeroValuesAreWritten() {
	this.expectFindOne(this.storedApp(`{}`))

	this.repository.
		On("Update", entity.App{ID: appId}, testifymock.MatchedBy(func(dao dto.AppUpdateDao) bool {
			return dao.Private != nil && !*dao.Private &&
				dao.TokenExpirationTime != nil && *dao.TokenExpirationTime == 0
		}), testifymock.Anything).
		Return(int64(1), nil)

	_, err := this.service.Update(appId, this.organization, &dto.UpdateApp{
		Private:             pointer(false),
		TokenExpirationTime: pointer(int64(0)),
	})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

func (this *AppUpdateTestSuite) TestMetadataIsMergedAndNeverOverwritten() {
	this.expectFindOne(this.storedApp(`{"kept": 1, "replaced": "old", "gone": true}`))

	this.repository.
		On("Update", entity.App{ID: appId}, testifymock.MatchedBy(func(dao dto.AppUpdateDao) bool {
			if dao.Metadata == nil {
				return false
			}

			merged := map[string]any{}
			_ = json.Unmarshal(*dao.Metadata, &merged)

			_, removed := merged["gone"]

			return merged["kept"] == float64(1) &&
				merged["replaced"] == "new" &&
				merged["added"] == float64(2) &&
				!removed
		}), testifymock.Anything).
		Return(int64(1), nil)

	patch := json.RawMessage(`{"replaced": "new", "added": 2, "gone": null}`)

	_, err := this.service.Update(appId, this.organization, &dto.UpdateApp{Metadata: &patch})

	assert.NoError(this.T(), err)
	this.repository.AssertExpectations(this.T())
}

// An empty body must not reach the repository: BuildUpdateMap would resolve to an
// empty map, Update would answer 0 rows affected, and that is indistinguishable
// from a missing row - which used to make a `{}` body a 404 on a row that exists.
func (this *AppUpdateTestSuite) TestAnEmptyPayloadTouchesNothingAndAnswersTheStoredRow() {
	stored := this.storedApp(`{"a": 1}`)
	this.expectFindOne(stored)

	updated, err := this.service.Update(appId, this.organization, &dto.UpdateApp{})

	assert.NoError(this.T(), err)
	assert.Equal(this.T(), stored.ID, updated.ID)
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// Folded into "does not exist", never 403: a distinguishable answer turns the id
// into an oracle for the apps of every other organization.
func (this *AppUpdateTestSuite) TestAnAppOfAnotherOrganizationIsNotFound() {
	other := otherOrgId
	stored := this.storedApp(`{}`)
	stored.OrganizationId = &other

	this.expectFindOne(stored)

	updated, err := this.service.Update(appId, this.organization, &dto.UpdateApp{Name: pointer("Renamed")})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("NOT_FOUND"), codeOf(this.T(), err))
	this.repository.AssertNotCalled(this.T(), "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func (this *AppUpdateTestSuite) TestANonObjectMetadataIsABadRequest() {
	this.expectFindOne(this.storedApp(`{}`))

	patch := json.RawMessage(`"not an object"`)

	updated, err := this.service.Update(appId, this.organization, &dto.UpdateApp{Metadata: &patch})

	assert.Nil(this.T(), updated)
	assert.Equal(this.T(), e.AppErrorCode("BAD_REQUEST"), codeOf(this.T(), err))
}

func TestAppUpdateSuite(t *testing.T) {
	suite.Run(t, new(AppUpdateTestSuite))
}
