package guards

import (
	"auth_service/common/constants"
	e "auth_service/common/errors"
	i "auth_service/common/interfaces"
	entity "auth_service/infra/entities"
	"slices"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var _ i.IGuard = &AppGuard{}

type OtpGuard struct {
	logger    *zap.Logger
	authGuard *AuthGuard
}

func NewOtpGuard(authGuard *AuthGuard, logger *zap.Logger) *OtpGuard {
	return &OtpGuard{
		authGuard: authGuard,
		logger:    logger,
	}
}

func (this *OtpGuard) Act(ctx *fiber.Ctx) error {
	this.logger.Info("Otp Guard Triggered")

	app, ok := ctx.Locals("app").(*entity.App)
	if !ok || app == nil {
		return e.ThrowInternalServerError("App context missing in AuthGuard - should be run before AuthGuard")
	}

	// Extract action from query params
	action := ctx.Query("action")

	if action == "" {
		return e.ThrowBadRequest("Action query parameter is required")
	}

	// If action is REGISTER, skip auth guard (allow unauthenticated)
	if slices.Contains(
		[]constants.AuthAction{
			constants.ActionRegister,
			constants.ActionLogin,
		},
		constants.AuthAction(action),
	) {
		return ctx.Next()
	}

	// For other actions, require authentication
	err := this.authGuard.Act(ctx)

	if err != nil {
		return err
	}

	return ctx.Next()
}
