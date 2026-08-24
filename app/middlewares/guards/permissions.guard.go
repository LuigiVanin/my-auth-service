package guards

import (
	e "auth_service/app/errors"
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
	"auth_service/shared/utils"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

/*
PermissionsGuard answers "may this caller reach this route", from what
permissions.Resolve leaves of the ceiling of the organization stacked with the
profile of the participation in it.

Resolving is what makes the model work: a participant of
{"api": {"*": {"methods": ["*"]}}} - what the owner of any organization holds -
collapses onto whatever its organization may do, and never onto more.

Row level visibility is NOT decided here. This answers "may this profile call GET
/core/apps", not "which apps may this user see". See docs/steering/service-layer.md.
*/
type PermissionsGuard struct {
	logger *zap.Logger
}

func NewPermissionsGuard(logger *zap.Logger) *PermissionsGuard {
	return &PermissionsGuard{logger: logger}
}

func (this *PermissionsGuard) Act(ctx fiber.Ctx) error {
	organization, ok := ctx.Locals("organization").(*entity.Organization)

	if !ok || organization == nil {
		return e.ThrowInternalServerError("Organization context missing in PermissionsGuard - should be run after OrganizationGuard")
	}

	participant, ok := ctx.Locals("participant").(*entity.Participant)

	if !ok || participant == nil {
		return e.ThrowInternalServerError("Participant context missing in PermissionsGuard - should be run after OrganizationGuard")
	}

	if organization.Profile == nil || participant.Profile == nil {
		return e.ThrowPermissionDeniedError("Profile not found for the organization or the participation")
	}

	resolved, err := permissions.Resolve(
		organization.Profile.Permissions,
		participant.Profile.Permissions,
	)

	if err != nil {
		this.logger.Error("Failed to resolve the permissions of the request", zap.Error(err))
		return e.ThrowInternalServerError("Failed to resolve permissions")
	}

	// Every denial carries what the request was matched against.
	denial := utils.JSON{
		"organization_permissions": organization.Profile.Permissions,
		"participant_permissions":  participant.Profile.Permissions,
		"resolved_permissions":     resolved.Api,
	}

	permission, hasPermission := this.findMatchingPermission(ctx.Route().Path, *resolved)

	if !hasPermission {
		this.logger.Warn("No matching permission found for path", zap.String("path", ctx.Route().Path))
		return e.ThrowPermissionDeniedError("User does not have permission to access this resource", denial)
	}

	if !this.isMethodAllowed(ctx.Method(), permission.Methods) {
		this.logger.Warn(
			"Method not allowed",
			zap.String("path", ctx.Route().Path),
			zap.String("method", ctx.Method()),
		)

		return e.ThrowPermissionDeniedError("User does not have permission to perform this action", denial)
	}

	if !this.isQueryAllowed(ctx, permission.Query) {
		this.logger.Warn(
			"Query parameters not allowed",
			zap.String("path", ctx.Route().Path),
			zap.Any("query", ctx.Queries()),
		)

		return e.ThrowPermissionDeniedError("User does not have permission with these query parameters", denial)
	}

	return ctx.Next()
}

// Matches ctx.Route().Path, the registered route pattern and never the request
// URL, so a key is written as `/core/apps/:id` and never `/core/apps/9f3c...`.
func (this *PermissionsGuard) findMatchingPermission(
	path string,
	resolved permissions.Resolved,
) (permissions.ResolvedRule, bool) {

	if rule, ok := resolved.Api[permissions.Wildcard]; ok {
		return rule, true
	}

	if rule, ok := resolved.Api[path]; ok {
		return rule, true
	}

	for key, rule := range resolved.Api {

		uuidRegexCode := `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

		regexKey := strings.ReplaceAll(key, ":id", `\d+`)
		regexKey = strings.ReplaceAll(regexKey, ":uuid", uuidRegexCode)

		regexKey = `^` + regexKey + `$`

		matched, err := regexp.MatchString(regexKey, path)
		if err == nil && matched {
			return rule, true
		}
	}

	return permissions.ResolvedRule{}, false
}

func (this *PermissionsGuard) isMethodAllowed(method string, allowedMethods []string) bool {
	for _, m := range allowedMethods {
		if m == permissions.Wildcard || strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// Denies any parameter the document does not mention, so adding a query parameter
// to a route means updating the profiles that reach it. A key carries a list of
// patterns because Resolve keeps the constraints of every layer; all must match.
func (this *PermissionsGuard) isQueryAllowed(ctx fiber.Ctx, allowedQuery map[string][]string) bool {
	if len(allowedQuery) == 0 {
		return true
	}

	queries := ctx.Queries()

	for key, value := range queries {
		patterns, ok := allowedQuery[key]

		if !ok {
			if patterns, ok = allowedQuery[permissions.Wildcard]; !ok {
				return false
			}
		}

		for _, pattern := range patterns {
			if pattern == permissions.Wildcard {
				continue
			}

			matched, err := regexp.MatchString(pattern, value)

			if err != nil {
				this.logger.Error(
					"Regex match failed for query parameter",
					zap.String("key", key),
					zap.String("regex", pattern),
					zap.Error(err),
				)

				return false
			}

			if !matched {
				return false
			}
		}
	}

	return true
}
