package routes_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	authcontroller "auth_service/app/modules/authorize/controller"
	avatarcontroller "auth_service/app/modules/avatar/controller"
	appcontroller "auth_service/app/modules/core/app/controller"
	orgcontroller "auth_service/app/modules/core/organization/controller"
	profilecontroller "auth_service/app/modules/core/profile/controller"
	usercontroller "auth_service/app/modules/core/user/controller"
	poolcontroller "auth_service/app/modules/core/user_pool/controller"
	logincontroller "auth_service/app/modules/login/controller"
	registercontroller "auth_service/app/modules/register/controller"
	entity "auth_service/infra/entities"
	"auth_service/shared/storage"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func registerEveryRoute(t *testing.T) (*fiber.App, *openapi.Builder) {
	t.Helper()

	server := fiber.New()
	builder := openapi.NewBuilder("test", "test", "test")
	logger := zap.NewNop()
	urls := storage.NewUrlResolver("https://cdn.test")

	appcontroller.NewAppController(nil, nil, nil, nil, nil, logger, builder, urls).Register(server)
	poolcontroller.NewUserPoolController(nil, nil, nil, nil, nil, builder, urls).Register(server)
	usercontroller.NewUserController(nil, nil, nil, nil, logger, builder, nil, urls).Register(server)
	orgcontroller.NewOrganizationController(nil, nil, nil, nil, nil, nil, logger, builder, urls).Register(server)
	profilecontroller.NewProfileController(nil, nil, nil, nil, nil, builder).Register(server)
	avatarcontroller.NewAvatarController(nil, nil, nil, nil, nil, urls, builder).Register(server)
	authcontroller.NewAuthorizeController(nil, nil, nil, builder, urls).Register(server)
	logincontroller.NewLoginController(nil, nil, logger, builder).Register(server)
	registercontroller.NewRegisterController(nil, nil, logger, builder).Register(server)

	return server, builder
}

// Mounted on the paths the grant catalog names - a path spelled differently in
// shared/permissions is a route no profile reaches.
func TestTheAvatarRoutesAreMountedWhereTheCatalogNamesThem(t *testing.T) {
	server, _ := registerEveryRoute(t)

	for _, subject := range []string{"/core/users/me", "/core/organizations/:id", "/core/apps/:id"} {
		assert.Contains(t, pathsOf(server, "POST"), subject+"/avatar/upload")
		assert.Contains(t, pathsOf(server, "PUT"), subject+"/avatar")
		assert.Contains(t, pathsOf(server, "DELETE"), subject+"/avatar")
	}
}

// avatar_url is built response by response, so forgetting it is easy: a new
// route answering a bare entity.User would ship without it. Any schema carrying
// every field of one of the three entities has to carry avatar_url too, nested
// ones included.
func TestEveryResponseCarryingAnAvatarOwnerCarriesAvatarUrl(t *testing.T) {
	_, builder := registerEveryRoute(t)

	encoded, err := json.Marshal(builder.Build())
	assert.NoError(t, err)

	var document any
	assert.NoError(t, json.Unmarshal(encoded, &document))

	owners := map[string][]string{
		"User":         serializedFields(entity.User{}),
		"Organization": serializedFields(entity.Organization{}),
		"App":          serializedFields(entity.App{}),
	}

	checked := 0

	walkSchemas(document, "", func(path string, properties map[string]any) {
		for name, fields := range owners {
			if !hasEvery(properties, fields) {
				continue
			}

			checked++
			assert.Contains(t, properties, "avatar_url", "%s at %s has no avatar_url", name, path)
		}
	})

	assert.Greater(t, checked, 20, "the walk found too few schemas - it is probably not reading the document")
}

// The fields encoding/json writes for an entity that are always present.
func serializedFields(value any) []string {
	fields := []string{}
	kind := reflect.TypeOf(value)

	for index := range kind.NumField() {
		name, options, _ := strings.Cut(kind.Field(index).Tag.Get("json"), ",")

		if name == "" || name == "-" || strings.Contains(options, "omitempty") {
			continue
		}

		fields = append(fields, name)
	}

	return fields
}

func hasEvery(properties map[string]any, fields []string) bool {
	for _, field := range fields {
		if _, ok := properties[field]; !ok {
			return false
		}
	}

	return true
}

func walkSchemas(node any, path string, visit func(path string, properties map[string]any)) {
	switch value := node.(type) {
	case map[string]any:
		if properties, ok := value["properties"].(map[string]any); ok {
			visit(path, properties)
		}

		for key, child := range value {
			walkSchemas(child, path+"/"+key, visit)
		}

	case []any:
		for _, child := range value {
			walkSchemas(child, path+"[]", visit)
		}
	}
}
