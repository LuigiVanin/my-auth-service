package middlewares_test

import (
	"encoding/json"
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

// newGuardedApp mounts the real PermissionsGuard behind a stand in for the
// OrganizationGuard, which is the only thing the guard reads from Locals. The
// routes mirror the ones a grant expands to, so ctx.Route().Path is the very
// string the permission document is keyed by.
func newGuardedApp(organization json.RawMessage, participant json.RawMessage) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(zap.NewNop()),
	})

	guard := guards.NewPermissionsGuard(zap.NewNop())

	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("organization", &entity.Organization{Profile: &entity.Profile{Permissions: organization}})
		ctx.Locals("participant", &entity.Participant{Profile: &entity.Profile{Permissions: participant}})

		return ctx.Next()
	})

	reached := func(ctx fiber.Ctx) error {
		return ctx.SendString("reached")
	}

	app.Get("/core/users", guard.Act, reached)
	app.Get("/core/users/:id", guard.Act, reached)
	app.Get("/core/apps", guard.Act, reached)

	return app
}

func call(t *testing.T, app *fiber.App, url string) int {
	t.Helper()

	resp, err := app.Test(httptest.NewRequest("GET", url, nil))

	assert.NoError(t, err)

	return resp.StatusCode
}

// The guard never learns that grants exist: the expansion happens inside the
// document, before Resolve intersects anything, so what arrives here is the same
// api shape it has always enforced.
func TestGuardEnforcesAGrantShapedProfile(t *testing.T) {
	app := newGuardedApp(
		json.RawMessage(`{"grants": ["as::users::READ", "as::apps::READ"]}`),
		json.RawMessage(`{"grants": ["as::users::READ"]}`),
	)

	assert.Equal(t, http.StatusOK, call(t, app, "/core/users"))
	assert.Equal(t, http.StatusOK, call(t, app, "/core/users/9f3c"))

	// The ceiling grants apps, the participant does not: no layer may widen the
	// one above it, and none may widen itself either.
	assert.Equal(t, http.StatusForbidden, call(t, app, "/core/apps"))
}

// A grant of the participation that the ceiling does not hold is clamped at
// request time, not only at write time.
func TestGuardClampsAGrantTheCeilingDoesNotHold(t *testing.T) {
	app := newGuardedApp(
		json.RawMessage(`{"grants": ["as::users::READ"]}`),
		json.RawMessage(`{"grants": ["as::users::READ", "as::apps::READ"]}`),
	)

	assert.Equal(t, http.StatusOK, call(t, app, "/core/users"))
	assert.Equal(t, http.StatusForbidden, call(t, app, "/core/apps"))
}

// api wins the whole path over the expansion of a grant, so it can narrow what
// the grant would have opened - here a query parameter the grant left free. The
// paths the api entry does not name keep coming from the grant.
func TestApiNarrowsAPathTheGrantWouldOpen(t *testing.T) {
	document := json.RawMessage(`{
		"grants": ["as::users::READ"],
		"api": {"/core/users": {"methods": ["GET"], "query": {"skip": "^[0-9]+$"}}}
	}`)

	app := newGuardedApp(document, document)

	assert.Equal(t, http.StatusOK, call(t, app, "/core/users?skip=10"))
	assert.Equal(t, http.StatusForbidden, call(t, app, "/core/users?skip=abc"))
	assert.Equal(t, http.StatusForbidden, call(t, app, "/core/users?limit=10"))

	assert.Equal(t, http.StatusOK, call(t, app, "/core/users/9f3c"))
}
