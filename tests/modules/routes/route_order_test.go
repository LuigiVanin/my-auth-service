// Package routes_test guards the one property of route registration that fiber
// cannot check and the compiler cannot see: a literal path has to be registered
// before a parameter path of the same method and shape, or the parameter route
// swallows it.
//
// The PermissionsGuard matches on ctx.Route().Path, so a swallowed literal does
// not 404 - it silently resolves to the wrong permission key and answers 403 to
// everyone holding the grant of the literal. See steering/guards-and-middlewares.md.
package routes_test

import (
	"encoding/json"
	"slices"
	"testing"

	authcontroller "auth_service/app/modules/authorize/controller"
	appcontroller "auth_service/app/modules/core/app/controller"
	orgcontroller "auth_service/app/modules/core/organization/controller"
	profilecontroller "auth_service/app/modules/core/profile/controller"
	usercontroller "auth_service/app/modules/core/user/controller"
	poolcontroller "auth_service/app/modules/core/user_pool/controller"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Every controller of the /core surface, registered on one server the way
// main.go does it. The guards are nil: Register only takes method values off
// them, and no request is ever sent.
func registerCoreRoutes(t *testing.T) *fiber.App {
	t.Helper()

	server := fiber.New()
	builder := openapi.NewBuilder("test", "test", "test")
	logger := zap.NewNop()

	appcontroller.NewAppController(nil, nil, nil, nil, nil, logger, builder).Register(server)
	poolcontroller.NewUserPoolController(nil, nil, nil, nil, nil, builder).Register(server)
	usercontroller.NewUserController(nil, nil, nil, nil, logger, builder, nil).Register(server)
	orgcontroller.NewOrganizationController(nil, nil, nil, nil, nil, nil, logger, builder).Register(server)
	profilecontroller.NewProfileController(nil, nil, nil, nil, nil, builder).Register(server)

	return server
}

func pathsOf(server *fiber.App, method string) []string {
	paths := []string{}

	for _, route := range server.GetRoutes(true) {
		if route.Method == method {
			paths = append(paths, route.Path)
		}
	}

	return paths
}

// assertRegisteredBefore fails when `first` does not come before `second` in the
// registration order of that method.
func assertRegisteredBefore(t *testing.T, server *fiber.App, method string, first string, second string) {
	t.Helper()

	paths := pathsOf(server, method)

	firstAt := slices.Index(paths, first)
	secondAt := slices.Index(paths, second)

	assert.NotEqual(t, -1, firstAt, "%s %s is not registered", method, first)
	assert.NotEqual(t, -1, secondAt, "%s %s is not registered", method, second)

	assert.Less(
		t, firstAt, secondAt,
		"%s %s has to be registered before %s %s, or the parameter route swallows the literal",
		method, first, method, second,
	)
}

// The expensive one: LOGIN_PROFILE holds as::organizations::switch::UPDATE and
// not as::organizations::UPDATE, so a swallowed `switch` takes the ability to
// change organization away from every signed up user.
func TestOrganizationSwitchIsNotSwallowedByTheUpdateRoute(t *testing.T) {
	server := registerCoreRoutes(t)

	assertRegisteredBefore(t, server, "PUT", "/core/organizations/switch", "/core/organizations/:id")
}

func TestUserMeRoutesComeBeforeTheParameterRoutes(t *testing.T) {
	server := registerCoreRoutes(t)

	assertRegisteredBefore(t, server, "GET", "/core/users/me", "/core/users/:id")
	assertRegisteredBefore(t, server, "PUT", "/core/users/me", "/core/users/:id")
}

// Already documented in the profile controller, pinned here so the whole rule
// lives in one test.
func TestGrantsIsNotSwallowedByTheProfileRoute(t *testing.T) {
	server := registerCoreRoutes(t)

	assertRegisteredBefore(t, server, "GET", "/core/grants", "/core/profiles/:id")
}

// The update routes this feature added are actually mounted, on the path the
// grant catalog names. A path written one way in shared/permissions and another
// way here is a route no profile can reach.
func TestTheUpdateRoutesAreMountedWhereTheCatalogNamesThem(t *testing.T) {
	server := registerCoreRoutes(t)

	for _, path := range []string{
		"/core/apps/:id",
		"/core/users/:id",
		"/core/users/me",
		"/core/users_pool/:id",
		"/core/organizations/:id",
		"/core/profiles/:id",
	} {
		assert.Contains(t, pathsOf(server, "PUT"), path)
	}
}

// Same property, on the /auth surface. It carries no parameter route, so nothing
// can be swallowed here - what this guards is the other half: a path spelled one
// way in shared/permissions and another way in the controller is a route no
// profile reaches, and nothing checks the two against each other.
func TestTheAuthRoutesAreMountedWhereTheCatalogNamesThem(t *testing.T) {
	server := fiber.New()
	builder := openapi.NewBuilder("test", "test", "test")

	authcontroller.NewAuthorizeController(nil, nil, nil, builder).Register(server)

	for method, path := range map[string]string{
		"POST": "/auth/revoke",
		"PUT":  "/auth/change_email",
	} {
		assert.Contains(t, pathsOf(server, method), path)
	}

	assert.Contains(t, pathsOf(server, "PUT"), "/auth/verify_email")

	document := builder.Build()
	encoded, err := json.Marshal(document)

	assert.NoError(t, err)
	assert.Contains(t, string(encoded), "/auth/revoke")
	assert.Contains(t, string(encoded), "/auth/change_email")
}

// The OpenAPI document is built once at boot, by an fx.Invoke that cmd/main_test
// does not run - fx.ValidateApp only type checks the graph. The update payloads
// are the first ones in this service with a *[]string and a *json.RawMessage, so
// the reflection behind AddBody is exercised here instead of at startup.
func TestTheOpenApiDocumentBuildsWithTheUpdatePayloads(t *testing.T) {
	server := fiber.New()
	builder := openapi.NewBuilder("test", "test", "test")
	logger := zap.NewNop()

	appcontroller.NewAppController(nil, nil, nil, nil, nil, logger, builder).Register(server)
	poolcontroller.NewUserPoolController(nil, nil, nil, nil, nil, builder).Register(server)
	usercontroller.NewUserController(nil, nil, nil, nil, logger, builder, nil).Register(server)
	orgcontroller.NewOrganizationController(nil, nil, nil, nil, nil, nil, logger, builder).Register(server)
	profilecontroller.NewProfileController(nil, nil, nil, nil, nil, builder).Register(server)

	document := builder.Build()

	assert.NotNil(t, document)

	encoded, err := json.Marshal(document)

	assert.NoError(t, err)
	assert.Contains(t, string(encoded), "/core/apps/{id}")
	assert.Contains(t, string(encoded), "/core/users/me")
	assert.Contains(t, string(encoded), "/core/users_pool/{id}")
	assert.Contains(t, string(encoded), "/core/organizations/{id}")
}
