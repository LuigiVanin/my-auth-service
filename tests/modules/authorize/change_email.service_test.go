package authorize_test

import (
	"encoding/json"
	"testing"

	"auth_service/app/modules/authorize/models"
	"auth_service/app/modules/authorize/services"
	odto "auth_service/app/modules/core/otp/models"
	udto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	repo "auth_service/shared/repository"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const newEmail = "moved@example.com"

type changeEmailMocks struct {
	otpService  *mock.MockOtpService
	userService *mock.MockUserService
	userRepo    *mock.MockUserRepository
	sessionRepo *mock.MockSessionRepository
	txManager   *mock.MockTransactionManager
}

func changeEmailOtp() *entity.Otp {
	metadata, _ := json.Marshal(odto.OtpStoredMetadata{
		Payload: odto.OtpStoredMetadataPayload{Email: newEmail},
	})

	return &entity.Otp{ID: otpId, Action: string(constants.ActionChangeEmail), Metadata: metadata}
}

// taken says whether the address the caller is moving to already belongs to
// somebody in the pool.
func newChangeEmailService(taken bool) (services.IAuthorizeService, *changeEmailMocks) {
	mocks := &changeEmailMocks{
		otpService:  new(mock.MockOtpService),
		userService: new(mock.MockUserService),
		userRepo:    new(mock.MockUserRepository),
		sessionRepo: new(mock.MockSessionRepository),
		txManager:   new(mock.MockTransactionManager),
	}

	mocks.otpService.
		On("ValidateConsumable", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(changeEmailOtp(), nil)
	mocks.otpService.On("Invalidate", testifymock.Anything).Return()

	mocks.userService.
		On("IsAlreadyCreated", testifymock.Anything, testifymock.Anything).
		Return(taken, nil)
	mocks.txManager.On("Tx").Return(&repo.Tx{}, nil)

	mocks.userRepo.
		On("Update", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)

	mocks.sessionRepo.
		On("RevokeAllForUser", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(int64(2), nil)

	service := services.NewAuthorizeService(
		nil, mocks.sessionRepo, nil, mocks.userRepo, mocks.userService,
		mocks.otpService, nil, nil, mocks.txManager, zap.NewNop(),
	)

	return service, mocks
}

func changeEmailBody() models.ChangeEmail {
	return models.ChangeEmail{
		Email: newEmail,
		Otp:   odto.PayloadOtpData{Id: otpId, Code: otpCode},
	}
}

func caller() *entity.User {
	return &entity.User{ID: 1, Email: storedEmail}
}

func TestChangeEmailWritesTheAddressAndVerifiesIt(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	updated, err := service.ChangeEmail(appWithPool(), caller(), changeEmailBody())

	assert.NoError(t, err)

	// The response carries the row as written, so the client does not have to guess
	// what the change did.
	assert.Equal(t, newEmail, updated.Email)
	assert.True(t, updated.VerifyEmail)
	assert.NotNil(t, updated.VerifiedEmailDate)

	dao := mocks.userRepo.Calls[0].Arguments.Get(1).(udto.UserUpdateDao)

	assert.NotNil(t, dao.Email)
	assert.Equal(t, newEmail, *dao.Email)
	assert.NotNil(t, dao.VerifyEmail)
	assert.True(t, *dao.VerifyEmail, "the code was delivered to the new address, so possession is proven")
	assert.NotNil(t, dao.VerifiedEmailDate)
}

// The requirement is that the caller comes out logged out, and that it happens in
// the same unit of work as the address moving.
func TestChangeEmailClosesEverySessionOfTheCaller(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	_, err := service.ChangeEmail(appWithPool(), caller(), changeEmailBody())

	assert.NoError(t, err)
	mocks.sessionRepo.AssertCalled(t, "RevokeAllForUser", uint(1), "app", testifymock.Anything)
}

// Both writes have to carry the transaction, or one of them silently runs on the
// pool and no rollback undoes it.
func TestChangeEmailWritesBothSidesOnTheSameTransaction(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	_, err := service.ChangeEmail(appWithPool(), caller(), changeEmailBody())
	assert.NoError(t, err)

	userOptions := mocks.userRepo.Calls[0].Arguments.Get(2).([]repo.Option)
	sessionOptions := mocks.sessionRepo.Calls[0].Arguments.Get(2).([]repo.Option)

	assert.Len(t, userOptions, 1)
	assert.Len(t, sessionOptions, 1)
	assert.NotNil(t, userOptions[0].Tx)
	assert.Same(t, userOptions[0].Tx, sessionOptions[0].Tx)
}

func TestChangeEmailConsumesTheOtp(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	_, err := service.ChangeEmail(appWithPool(), caller(), changeEmailBody())

	assert.NoError(t, err)
	mocks.otpService.AssertCalled(t, "Invalidate", otpId)
}

// The address is checked again here because it can be taken between generating
// the code and spending it.
func TestChangeEmailRefusesAnAddressTakenInTheMeantime(t *testing.T) {
	service, mocks := newChangeEmailService(true)

	_, err := service.ChangeEmail(appWithPool(), caller(), changeEmailBody())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already uses this email")

	mocks.userRepo.AssertNotCalled(t, "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
	mocks.sessionRepo.AssertNotCalled(t, "RevokeAllForUser", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func TestChangeEmailRefusesTheAddressAlreadyOnTheAccount(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	current := caller()
	current.Email = newEmail

	_, err := service.ChangeEmail(appWithPool(), current, changeEmailBody())

	assert.Error(t, err)
	mocks.userRepo.AssertNotCalled(t, "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

// The OTP proves possession of the address stored inside it, so a body naming a
// different one is not that proof.
func TestChangeEmailRefusesAnOtpStoredForAnotherAddress(t *testing.T) {
	service, mocks := newChangeEmailService(false)

	payload := changeEmailBody()
	payload.Email = "elsewhere@example.com"

	_, err := service.ChangeEmail(appWithPool(), caller(), payload)

	assert.Error(t, err)
	mocks.userRepo.AssertNotCalled(t, "Update", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}
