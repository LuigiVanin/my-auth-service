package guards

import (
	e "auth_service/app/errors"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ApiPermission struct {
	Query   map[string]string `json:"query"`
	Methods []string          `json:"methods"`
}

type Permission struct {
	Api map[string]ApiPermission `json:"api"`
}

/*
This is an example of the permission field on the profiles table:

```json
// MANAGER profile

	{
	  "api": {
	    "/core/apps": {
	      "methods": ["POST", "GET"]
	    },
	    "/core/apps/:id": {
	      "methods": ["GET", "PUT", "DELETE"]
	    },
	    "/core/apps/:id/users": {
	      "methods": ["GET", "POST"],
	      "query": {
	        "skip": "^[0-9]+$",
	        "limit": "^[0-9]+$",
	        "name": "^.+$",
	        "email": "^.+$"
	      }
	    }
	  }
	}

```

The example bellow is also a permission, but this time from the ADMIN profile, where there is the use
of the wild card:
```json

	{
	  "api": {
	    "*": {
	      "methods": ["*"]
	    },
		}
	}

```

OBS.: The default behaviour for a empty query field is to assume: "query": { "*" : "*" }
*/
type PermissionsGuard struct {
	logger *zap.Logger
}

func NewPermissionsGuard(logger *zap.Logger) *PermissionsGuard {
	return &PermissionsGuard{logger: logger}
}

// Act checks if the current user has the necessary permissions to access the requested resource
func (this *PermissionsGuard) Act(ctx *fiber.Ctx) error {
	this.logger.Info("Permissions Guard Triggered")

	app, ok := ctx.Locals("app").(*entity.App)
	if !ok || app == nil {
		return e.ThrowInternalServerError("App context missing in PermissionsGuard - should be run after AppGuard")
	}

	user, ok := ctx.Locals("user").(*entity.User)

	if !ok || user == nil {
		return e.ThrowInternalServerError("User context missing in PermissionsGuard - should be run after AuthGuard")
	}

	if user.Profile == nil {
		return e.ThrowPermissionDeniedError("User profile not found")
	}

	if user.Profile.Permissions == nil {
		return e.ThrowPermissionDeniedError("User profile permissions not found")
	}

	var permissions Permission

	// Unmarshal the permissions into the Permission struct
	err := json.Unmarshal(user.Profile.Permissions, &permissions)

	if err != nil {
		this.logger.Error("Failed to unmarshal permissions", zap.Error(err), zap.String("permissions", string(user.Profile.Permissions)))
		return e.ThrowInternalServerError("Failed to unmarshal permissions")
	}

	permissionJson := utils.JSON{"api": permissions.Api}

	// Check for a match in permissions
	apiPerm, hasPerm := this.findMatchingPermission(ctx.Route().Path, permissions)

	fmt.Print("hasPerm: ", hasPerm)
	if !hasPerm {
		this.logger.Warn("No matching permission found for path", zap.String("path", ctx.Route().Path))
		return e.ThrowPermissionDeniedError(
			"User does not have permission to access this resource",
			permissionJson,
		)
	}

	// Check method
	if !this.isMethodAllowed(ctx.Method(), apiPerm.Methods) {
		this.logger.Warn("Method not allowed", zap.String("path", ctx.Route().Path), zap.String("method", ctx.Method()))
		return e.ThrowPermissionDeniedError(
			"User does not have permission to perform this action",
			permissionJson,
		)
	}

	// Check query parameters
	if !this.isQueryAllowed(ctx, apiPerm.Query) {
		this.logger.Warn("Query parameters not allowed", zap.String("path", ctx.Route().Path), zap.Any("query", ctx.Queries()))
		return e.ThrowPermissionDeniedError(
			"User does not have permission with these query parameters",
			permissionJson,
		)
	}

	this.logger.Info("Permissions Guard passed", zap.String("path", ctx.Route().Path), zap.String("method", ctx.Method()), zap.Any("query", ctx.Queries()))

	return ctx.Next()
}

func (this *PermissionsGuard) findMatchingPermission(path string, permissions Permission) (ApiPermission, bool) {

	if perm, ok := permissions.Api["*"]; ok {
		return perm, true
	}

	if perm, ok := permissions.Api[path]; ok {
		return perm, true
	}

	for key, perm := range permissions.Api {

		uuidRegexCode := `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

		regexKey := strings.ReplaceAll(key, ":id", `\d+`)
		regexKey = strings.ReplaceAll(regexKey, ":uuid", uuidRegexCode)

		regexKey = `^` + regexKey + `$`

		matched, err := regexp.MatchString(regexKey, path)
		if err == nil && matched {
			return perm, true
		}
	}

	return ApiPermission{}, false
}

func (this *PermissionsGuard) isMethodAllowed(method string, allowedMethods []string) bool {
	for _, m := range allowedMethods {
		if m == "*" || strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

func (this *PermissionsGuard) isQueryAllowed(ctx *fiber.Ctx, allowedQuery map[string]string) bool {
	if len(allowedQuery) == 0 {
		return true
	}

	queries := ctx.Queries()

	if val, ok := allowedQuery["*"]; ok && val == "*" {
		return true
	}

	for key, value := range queries {
		regexStr, ok := allowedQuery[key]
		if !ok {
			if wildcardRegex, hasWildcard := allowedQuery["*"]; hasWildcard {
				regexStr = wildcardRegex
			} else {
				return false
			}
		}

		if regexStr == "*" {
			continue
		}

		matched, err := regexp.MatchString(regexStr, value)
		if err != nil {
			this.logger.Error("Regex match failed for query parameter", zap.String("key", key), zap.String("regex", regexStr), zap.Error(err))
			return false
		}
		if !matched {
			return false
		}
	}

	return true
}
