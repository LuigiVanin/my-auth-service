package avatar_test

import (
	"errors"
	"testing"

	e "auth_service/app/errors"
	"auth_service/app/modules/avatar/services"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"
	"auth_service/shared/storage"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	organizationId = "11111111-1111-1111-1111-111111111111"
	appId          = "33333333-3333-3333-3333-333333333333"

	pendingKey  = "pending/avatar/users/42/9f1c3a7e2b4d6081.webp"
	promotedKey = "avatar/users/42/9f1c3a7e2b4d6081.webp"
	previousKey = "avatar/users/42/00000000aaaaaaaa.png"
)

type AvatarServiceTestSuite struct {
	suite.Suite
	storage             *mock.MockStorage
	userService         *mock.MockUserService
	organizationService *mock.MockOrganizationService
	appService          *mock.MockAppService
	service             *services.AvatarService

	user         *entity.User
	organization *entity.Organization

	calls []string
}

func (this *AvatarServiceTestSuite) SetupTest() {
	this.storage = new(mock.MockStorage)
	this.userService = new(mock.MockUserService)
	this.organizationService = new(mock.MockOrganizationService)
	this.appService = new(mock.MockAppService)
	this.service = this.build(this.storage)

	this.user = &entity.User{ID: 42}
	this.organization = &entity.Organization{ID: organizationId}
	this.calls = nil
}

func (this *AvatarServiceTestSuite) build(files storage.IStorage) *services.AvatarService {
	return services.NewAvatarService(
		files,
		this.userService,
		this.organizationService,
		this.appService,
		&config.Config{},
		zap.NewNop(),
	)
}

func (this *AvatarServiceTestSuite) record(name string) func(testifymock.Arguments) {
	return func(testifymock.Arguments) {
		this.calls = append(this.calls, name)
	}
}

func (this *AvatarServiceTestSuite) storedImage() {
	this.storage.On("Head", pendingKey).Return(&storage.Object{Size: 120_000, ContentType: "image/webp"}, nil)
}

func codeOf(t assert.TestingT, err error) e.AppErrorCode {
	appError, ok := err.(*e.AppError)

	if !assert.True(t, ok, "expected *AppError, got %T", err) {
		return ""
	}

	return appError.Code.First
}

func TestAvatarServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AvatarServiceTestSuite))
}

func (this *AvatarServiceTestSuite) TestRequestSignsAKeyOfTheCaller() {
	this.storage.
		On("PresignUpload", testifymock.Anything, "image/webp", testifymock.Anything).
		Return(&storage.UploadGrant{Key: "signed"}, nil)

	grant, err := this.service.RequestUserUpload(this.user, "image/webp")

	assert.NoError(this.T(), err)
	assert.Equal(this.T(), "signed", grant.Key)

	signed := this.storage.Calls[0].Arguments.String(0)
	assert.Regexp(this.T(), `^pending/avatar/users/42/[0-9a-f]{16}\.webp$`, signed)
}

func (this *AvatarServiceTestSuite) TestRequestRefusesAnUnacceptedType() {
	_, err := this.service.RequestUserUpload(this.user, "image/svg+xml")

	assert.Equal(this.T(), e.AppErrorCode("BAD_REQUEST"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "PresignUpload", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func (this *AvatarServiceTestSuite) TestEveryRouteAnswers501WithoutStorage() {
	service := this.build(storage.Unconfigured{})

	_, requestErr := service.RequestUserUpload(this.user, "image/webp")
	_, confirmErr := service.ConfirmUser(this.user, this.organization, pendingKey)
	_, removeErr := service.RemoveOrganization(organizationId, this.organization)
	_, appErr := service.ConfirmApp(appId, this.organization, pendingKey)

	for _, err := range []error{requestErr, confirmErr, removeErr, appErr} {
		assert.Equal(this.T(), e.AppErrorCode("NOT_IMPLEMENTED"), codeOf(this.T(), err))
	}
}

func (this *AvatarServiceTestSuite) TestConfirmRefusesAKeyThatIsNotTheCallers() {
	_, err := this.service.ConfirmUser(this.user, this.organization, "pending/avatar/users/57/9f1c3a7e2b4d6081.webp")

	assert.Equal(this.T(), e.AppErrorCode("BAD_REQUEST"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "Head", testifymock.Anything)
}

// The case RC-3 exists for: a key already published under another owner would
// be deleted by the next swap of the caller.
func (this *AvatarServiceTestSuite) TestConfirmRefusesAPublishedKey() {
	_, err := this.service.ConfirmUser(this.user, this.organization, "avatar/organizations/other/9f1c3a7e2b4d6081.webp")

	assert.Equal(this.T(), e.AppErrorCode("BAD_REQUEST"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "Head", testifymock.Anything)
}

func (this *AvatarServiceTestSuite) TestConfirmAnswers404WhenNothingWasUploaded() {
	this.storage.On("Head", pendingKey).Return(nil, nil)

	_, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.Equal(this.T(), e.AppErrorCode("NOT_FOUND"), codeOf(this.T(), err))
}

func (this *AvatarServiceTestSuite) TestConfirmAnswers422WhenTheFileIsNotWhatWasSigned() {
	this.storage.On("Head", pendingKey).Return(&storage.Object{Size: 3 * 1024 * 1024, ContentType: "image/webp"}, nil)

	_, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.Equal(this.T(), e.AppErrorCode("UNPROCESSABLE_ENTITY"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "Copy", testifymock.Anything, testifymock.Anything)
}

func (this *AvatarServiceTestSuite) TestConfirmAnswers500WhenTheStorageFails() {
	this.storage.On("Head", pendingKey).Return(nil, errors.New("timeout"))

	_, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.Equal(this.T(), e.AppErrorCode("INTERNAL_SERVER_ERROR"), codeOf(this.T(), err))
}

func (this *AvatarServiceTestSuite) TestConfirmPromotesWritesThenCleansUp() {
	previous := previousKey
	this.user.AvatarKey = &previous
	updated := &entity.User{ID: 42}

	this.storedImage()
	this.storage.On("Copy", pendingKey, promotedKey).Return(nil).Run(this.record("copy"))
	this.userService.On("SetAvatarKey", uint(42), testifymock.MatchedBy(func(key *string) bool {
		return key != nil && *key == promotedKey
	})).Return(nil).Run(this.record("write"))
	this.storage.On("Delete", pendingKey).Return(nil).Run(this.record("delete pending"))
	this.storage.On("Delete", previousKey).Return(nil).Run(this.record("delete previous"))
	this.userService.On("FindById", this.user, this.organization, uint(42)).Return(updated, nil)

	user, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.NoError(this.T(), err)
	assert.Same(this.T(), updated, user)
	assert.Equal(this.T(), []string{"copy", "write", "delete pending", "delete previous"}, this.calls)
}

// RC-5: the database already points at the new avatar, so the old one left
// behind is an orphan, not a failure.
func (this *AvatarServiceTestSuite) TestConfirmSucceedsWhenThePreviousAvatarCannotBeDeleted() {
	previous := previousKey
	this.user.AvatarKey = &previous

	this.storedImage()
	this.storage.On("Copy", pendingKey, promotedKey).Return(nil)
	this.userService.On("SetAvatarKey", uint(42), testifymock.Anything).Return(nil)
	this.storage.On("Delete", pendingKey).Return(nil)
	this.storage.On("Delete", previousKey).Return(errors.New("access denied"))
	this.userService.On("FindById", this.user, this.organization, uint(42)).Return(this.user, nil)

	_, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.NoError(this.T(), err)
}

func (this *AvatarServiceTestSuite) TestConfirmDiscardsThePromotedCopyWhenTheWriteFails() {
	this.storedImage()
	this.storage.On("Copy", pendingKey, promotedKey).Return(nil)
	this.userService.On("SetAvatarKey", uint(42), testifymock.Anything).Return(e.ThrowInternalServerError("boom"))
	this.storage.On("Delete", promotedKey).Return(nil)

	_, err := this.service.ConfirmUser(this.user, this.organization, pendingKey)

	assert.Equal(this.T(), e.AppErrorCode("INTERNAL_SERVER_ERROR"), codeOf(this.T(), err))
	this.storage.AssertCalled(this.T(), "Delete", promotedKey)
	this.storage.AssertNotCalled(this.T(), "Delete", pendingKey)
}

func (this *AvatarServiceTestSuite) TestRemoveWithoutAnAvatarAnswersTheSameBody() {
	this.userService.On("FindById", this.user, this.organization, uint(42)).Return(this.user, nil)

	user, err := this.service.RemoveUser(this.user, this.organization)

	assert.NoError(this.T(), err)
	assert.Same(this.T(), this.user, user)
	this.userService.AssertNotCalled(this.T(), "SetAvatarKey", testifymock.Anything, testifymock.Anything)
	this.storage.AssertNotCalled(this.T(), "Delete", testifymock.Anything)
}

func (this *AvatarServiceTestSuite) TestRemoveClearsTheKeyThenDeletesTheObject() {
	previous := previousKey
	this.user.AvatarKey = &previous

	this.userService.On("SetAvatarKey", uint(42), (*string)(nil)).Return(nil).Run(this.record("write"))
	this.storage.On("Delete", previousKey).Return(nil).Run(this.record("delete"))
	this.userService.On("FindById", this.user, this.organization, uint(42)).Return(this.user, nil)

	_, err := this.service.RemoveUser(this.user, this.organization)

	assert.NoError(this.T(), err)
	assert.Equal(this.T(), []string{"write", "delete"}, this.calls)
}

func (this *AvatarServiceTestSuite) TestAnotherOrganizationIsRefusedLikeItsUpdateRoute() {
	_, err := this.service.RequestOrganizationUpload("22222222-2222-2222-2222-222222222222", this.organization, "image/png")

	assert.Equal(this.T(), e.AppErrorCode("PERMISSION_DENIED"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "PresignUpload", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func (this *AvatarServiceTestSuite) TestTheOrganizationKeyIsScopedToTheOrganization() {
	key := "pending/avatar/organizations/" + organizationId + "/9f1c3a7e2b4d6081.png"
	promoted := "avatar/organizations/" + organizationId + "/9f1c3a7e2b4d6081.png"

	this.storage.On("Head", key).Return(&storage.Object{Size: 10, ContentType: "image/png"}, nil)
	this.storage.On("Copy", key, promoted).Return(nil)
	this.organizationService.On("SetAvatarKey", organizationId, testifymock.Anything).Return(nil)
	this.storage.On("Delete", key).Return(nil)
	this.organizationService.On("FindById", organizationId).Return(this.organization, nil)

	_, err := this.service.ConfirmOrganization(organizationId, this.organization, key)

	assert.NoError(this.T(), err)

	_, err = this.service.ConfirmOrganization(organizationId, this.organization, pendingKey)
	assert.Equal(this.T(), e.AppErrorCode("BAD_REQUEST"), codeOf(this.T(), err), "a user key is not an organization key")
}

// AppService.FindById carries the scope rule, so an app of another organization
// is a 404 before the storage is touched.
func (this *AvatarServiceTestSuite) TestAnAppOfAnotherOrganizationIsNotFound() {
	this.appService.On("FindById", appId, this.organization).Return(nil, e.ThrowNotFound("App not found in the current organization"))

	_, err := this.service.RequestAppUpload(appId, this.organization, "image/png")

	assert.Equal(this.T(), e.AppErrorCode("NOT_FOUND"), codeOf(this.T(), err))
	this.storage.AssertNotCalled(this.T(), "PresignUpload", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}
