package guards

import (
	e "auth_service/app/errors"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	i "auth_service/shared/interfaces"
	"slices"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

var _ i.IGuard = &OtpGuard{}

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

// Authenticate composes AuthGuard.Authenticate, not AuthGuard.Act: the chain is
// continued once, by Act below.
func (this *OtpGuard) Authenticate(ctx fiber.Ctx) error {
	this.logger.Info("Otp Guard Triggered")

	app, ok := ctx.Locals("app").(*entity.App)
	if !ok || app == nil {
		return e.ThrowInternalServerError("App context missing in OtpGuard - should be run after AppGuard")
	}

	// Extract action from query params
	action := ctx.Query("action")

	if action == "" {
		return e.ThrowBadRequest("Action query parameter is required")
	}

	// If action is REGISTER, LOGIN or FORGOT_PASSWORD skip auth guard (allow unauthenticated)
	if slices.Contains(
		[]constants.AuthAction{
			constants.ActionRegister,
			constants.ActionLogin,
			constants.ActionForgotPassword,
		},
		constants.AuthAction(action),
	) {
		return nil
	}

	// For other actions, require authentication
	return this.authGuard.Authenticate(ctx)
}

func (this *OtpGuard) Act(ctx fiber.Ctx) error {
	if err := this.Authenticate(ctx); err != nil {
		return err
	}

	return ctx.Next()
}
