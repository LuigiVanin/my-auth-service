package middlewares_test

import (
	"io"
	"net/http/httptest"
	"testing"

	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/authorize/models"
	entity "auth_service/infra/entities"
	smodels "auth_service/shared/models"
	"auth_service/tests/modules/mock"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Every guard and handler after the AuthGuard asserts ctx.Locals("user") as
// *entity.User. The authorize response carries a view for the client, so the
// guard has to unwrap it; storing the view compiles and fails every route.
func TestTheAuthGuardLeavesTheEntityInLocals(t *testing.T) {
	authService := new(mock.MockAuthorizeService)
	authService.
		On("Authorize", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(&dto.AuthorizeResponse{
			User:       smodels.UserView{User: entity.User{ID: 42, VerifyEmail: true}},
			SessionId:  "session",
			Authorized: true,
		}, nil)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.NewErrorHandler(zap.NewNop())})

	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("app", &entity.App{VerifyEmail: true})
		return ctx.Next()
	})

	var seen *entity.User

	app.Get(
		"/core/anything",
		guards.NewAuthGuard(authService, zap.NewNop()).Act,
		guards.NewEmailVerificationGuard(zap.NewNop()).Act,
		func(ctx fiber.Ctx) error {
			seen, _ = ctx.Locals("user").(*entity.User)
			return ctx.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest("GET", "/core/anything", nil)
	req.Header.Set("Authorization", "Bearer token")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	if assert.NotNil(t, seen, "Locals(\"user\") is not an *entity.User") {
		assert.Equal(t, uint(42), seen.ID)
	}
}

// A value of the wrong type is not a guard out of order, and saying "missing"
// sends whoever reads the 500 after the wrong cause.
func TestAWrongTypeInLocalsIsReportedAsSuch(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.NewErrorHandler(zap.NewNop())})

	app.Get(
		"/core/anything",
		func(ctx fiber.Ctx) error {
			ctx.Locals("app", &entity.App{})
			ctx.Locals("user", &smodels.UserView{})
			return ctx.Next()
		},
		guards.NewEmailVerificationGuard(zap.NewNop()).Act,
		func(ctx fiber.Ctx) error { return ctx.SendStatus(fiber.StatusOK) },
	)

	resp, err := app.Test(httptest.NewRequest("GET", "/core/anything", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	assert.Contains(t, string(body), `holds *models.UserView, expected *entity.User - AuthGuard stored the wrong type`)
	assert.NotContains(t, string(body), "missing")
}
