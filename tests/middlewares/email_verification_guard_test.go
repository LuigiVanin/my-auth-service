package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	entity "auth_service/infra/entities"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// newVerificationApp chains the guard the way a controller does - after the stand
// in for AuthGuard, never mounted by prefix. Mounted by prefix it would run before
// the route level AuthGuard and never find `user` in Locals, which is what the
// 500 below pins.
func newVerificationApp(appRequires bool, userVerified bool, mountByPrefix bool) *fiber.App {
	server := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(zap.NewNop()),
	})

	guard := guards.NewEmailVerificationGuard(zap.NewNop())

	server.Use("/core", func(ctx fiber.Ctx) error {
		ctx.Locals("app", &entity.App{VerifyEmail: appRequires})

		return ctx.Next()
	})

	if mountByPrefix {
		server.Use("/core", guard.Act)
	}

	authGuardStandIn := func(ctx fiber.Ctx) error {
		ctx.Locals("user", &entity.User{VerifyEmail: userVerified})

		return ctx.Next()
	}

	reached := func(ctx fiber.Ctx) error {
		return ctx.SendString("reached")
	}

	if mountByPrefix {
		server.Get("/core/users", authGuardStandIn, reached)
	} else {
		server.Get("/core/users", authGuardStandIn, guard.Act, reached)
	}

	return server
}

func callCore(t *testing.T, server *fiber.App) int {
	t.Helper()

	resp, err := server.Test(httptest.NewRequest("GET", "/core/users", nil))

	assert.NoError(t, err)

	return resp.StatusCode
}

func TestAnUnverifiedUserIsRefusedWhenTheAppRequiresVerification(t *testing.T) {
	assert.Equal(t, http.StatusForbidden, callCore(t, newVerificationApp(true, false, false)))
}

func TestAVerifiedUserPasses(t *testing.T) {
	assert.Equal(t, http.StatusOK, callCore(t, newVerificationApp(true, true, false)))
}

// The flag is the application's: a pool that does not require verification lets
// an unverified user through.
func TestAnUnverifiedUserPassesWhenTheAppDoesNotRequireVerification(t *testing.T) {
	assert.Equal(t, http.StatusOK, callCore(t, newVerificationApp(false, false, false)))
}

// Mounted by prefix the guard runs before the route level AuthGuard, so `user` is
// never in Locals and every /core route answers 500 - verified user included.
func TestMountingTheGuardByPrefixBreaksEveryRoute(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, callCore(t, newVerificationApp(true, true, true)))
}
