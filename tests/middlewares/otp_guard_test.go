package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/authorize/models"
	entity "auth_service/infra/entities"
	mock "auth_service/tests/modules/mock"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// newOtpGuardedApp mounts the real OtpGuard over the real AuthGuard, which is the
// composition the guard exists for, behind a stand in for the AppGuard - the only
// thing either of them reads from Locals.
func newOtpGuardedApp(authService *mock.MockAuthorizeService) (*fiber.App, *int) {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(zap.NewNop()),
	})

	authGuard := guards.NewAuthGuard(authService, zap.NewNop())
	otpGuard := guards.NewOtpGuard(authGuard, zap.NewNop())

	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("app", &entity.App{})

		return ctx.Next()
	})

	reached := 0

	app.Post("/otp/generate_consumable", otpGuard.Act, func(ctx fiber.Ctx) error {
		reached++

		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"otp_id": "1"})
	})

	return app, &reached
}

func callOtp(t *testing.T, app *fiber.App, action string, token string) *http.Response {
	t.Helper()

	req := httptest.NewRequest("POST", "/otp/generate_consumable?action="+action, nil)

	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := app.Test(req)

	assert.NoError(t, err)

	return resp
}

func authorized() *mock.MockAuthorizeService {
	service := new(mock.MockAuthorizeService)

	service.
		On("Authorize", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&dto.AuthorizeResponse{
			User:       entity.User{},
			SessionId:  "session",
			TokenType:  "access",
			Authorized: true,
		}, nil)

	return service
}

// The guard composes AuthGuard.Authenticate and continues the chain once. When it
// called AuthGuard.Act instead, the handler ran inside it and the second
// ctx.Next() walked past the last handler, so a request that had already created
// the OTP and sent the email was answered 404.
func TestAuthenticatedActionReachesTheHandlerOnce(t *testing.T) {
	app, reached := newOtpGuardedApp(authorized())

	resp := callOtp(t, app, "VERIFY_EMAIL", "Bearer token")

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, *reached)
}

func TestUnauthenticatedActionSkipsTheAuthGuard(t *testing.T) {
	authService := authorized()

	app, reached := newOtpGuardedApp(authService)

	resp := callOtp(t, app, "REGISTER", "")

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, *reached)

	authService.AssertNotCalled(t, "Authorize", testifymock.Anything, testifymock.Anything, testifymock.Anything)
}

func TestAuthenticatedActionWithoutATokenIsRefused(t *testing.T) {
	app, reached := newOtpGuardedApp(authorized())

	resp := callOtp(t, app, "VERIFY_EMAIL", "")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, 0, *reached)
}

func TestAMissingActionIsRefused(t *testing.T) {
	app, reached := newOtpGuardedApp(authorized())

	resp := callOtp(t, app, "", "Bearer token")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, 0, *reached)
}
