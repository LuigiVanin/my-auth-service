package guards

import (
	e "auth_service/app/errors"
	entity "auth_service/infra/entities"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type PermissionsGuard struct {
	logger *zap.Logger
}

func NewPermissionsGuard(logger *zap.Logger) *PermissionsGuard {
	return &PermissionsGuard{logger: logger}
}

// TODO: Implement permissions guard based on the permissions json found on the profile
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

	if strings.Contains(ctx.Route().Path, "/core") {
		if user.Profile == nil {
			return e.ThrowUnauthorizedError("User profile not found")
		}

		if user.Profile.Key == "ADMIN" || user.Profile.Key == "MANAGER" {
			return ctx.Next()
		} else {
			return e.ThrowUnauthorizedError("User does not have permission to access this resource")
		}
	} else {
		return ctx.Next()
	}
}
