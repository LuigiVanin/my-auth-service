package authorize_test

import (
	"encoding/json"
	"testing"
	"time"

	"auth_service/app/modules/authorize/models"
	"auth_service/app/modules/authorize/services"
	odto "auth_service/app/modules/core/otp/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/storage"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	storedEmail = "someone@example.com"
	otpId       = "0f1a4e1c-3b2d-4f5a-9c8b-7d6e5f4a3b2c"
	otpCode     = "123456"
)

type verifyEmailMocks struct {
	otpService  *mock.MockOtpService
	userService *mock.MockUserService
}

func otpForEmail(email string) *entity.Otp {
	metadata, _ := json.Marshal(odto.OtpStoredMetadata{
		Payload: odto.OtpStoredMetadataPayload{Email: email},
	})

	return &entity.Otp{ID: otpId, Action: string(constants.ActionVerifyEmail), Metadata: metadata}
}

func newVerifyEmailService(stored *entity.User) (services.IAuthorizeService, *verifyEmailMocks) {
	mocks := &verifyEmailMocks{
		otpService:  new(mock.MockOtpService),
		userService: new(mock.MockUserService),
	}

	mocks.otpService.
		On("ValidateConsumable", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(otpForEmail(storedEmail), nil)
	mocks.otpService.On("Invalidate", testifymock.Anything).Return()

	mocks.userService.
		On("FindUserInPool", testifymock.Anything, testifymock.Anything).
		Return(stored, nil)
	mocks.userService.
		On("Update", testifymock.Anything, testifymock.Anything).
		Return(int64(1), nil)

	service := services.NewAuthorizeService(
		nil, nil, nil, nil, mocks.userService, mocks.otpService, nil, nil, nil, zap.NewNop(),
		storage.NewUrlResolver("https://cdn.test"),
	)

	return service, mocks
}

func verifyEmailBody() models.VerifyEmailBody {
	return models.VerifyEmailBody{
		Email: storedEmail,
		Otp:   odto.PayloadOtpData{Id: otpId, Code: otpCode},
	}
}

func appWithPool() *entity.App {
	return &entity.App{ID: "app", UsersPool: entity.UsersPool{ID: "pool"}}
}

// The condition used to be `if !user.VerifyEmail { return nil }`, which skipped
// the write for exactly the users the route exists to serve.
func TestVerifyEmailWritesTheFlagForAnUnverifiedUser(t *testing.T) {
	stored := &entity.User{ID: 1, Email: storedEmail, VerifyEmail: false}

	service, mocks := newVerifyEmailService(stored)

	verified, err := service.VerifyEmail(appWithPool(), stored, verifyEmailBody())

	assert.NoError(t, err)
	assert.NotNil(t, verified)
	assert.True(t, verified.VerifyEmail, "the response has to carry the row that was just written")
	assert.NotNil(t, verified.VerifiedEmailDate)

	mocks.userService.AssertCalled(t, "Update", entity.User{ID: 1}, testifymock.AnythingOfType("models.UserUpdateDao"))
}

func TestVerifyEmailIsANoopForAnAlreadyVerifiedUser(t *testing.T) {
	now := time.Now()
	stored := &entity.User{ID: 1, Email: storedEmail, VerifyEmail: true, VerifiedEmailDate: &now}

	service, mocks := newVerifyEmailService(stored)

	verified, err := service.VerifyEmail(appWithPool(), stored, verifyEmailBody())

	assert.NoError(t, err)
	assert.NotNil(t, verified)
	mocks.userService.AssertNotCalled(t, "Update", testifymock.Anything, testifymock.Anything)
}

// ResetPassword consumed its OTP and this one did not, so the code stayed usable
// for the rest of its five minute life.
func TestVerifyEmailConsumesTheOtp(t *testing.T) {
	stored := &entity.User{ID: 1, Email: storedEmail}

	service, mocks := newVerifyEmailService(stored)

	_, err := service.VerifyEmail(appWithPool(), stored, verifyEmailBody())

	assert.NoError(t, err)
	mocks.otpService.AssertCalled(t, "Invalidate", otpId)
}

func TestVerifyEmailRefusesAnOtpStoredForAnotherEmail(t *testing.T) {
	stored := &entity.User{ID: 1, Email: storedEmail}

	service, _ := newVerifyEmailService(stored)

	payload := verifyEmailBody()
	payload.Email = "someone.else@example.com"

	_, err := service.VerifyEmail(appWithPool(), stored, payload)

	assert.Error(t, err)
}
