package guards

import (
	e "auth_service/app/errors"
	entity "auth_service/infra/entities"

	i "auth_service/shared/interfaces"
	"auth_service/shared/utils"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

var _ i.IGuard = &EmailVerificationGuard{}

type EmailVerificationGuard struct {
	logger *zap.Logger
}

func NewEmailVerificationGuard(logger *zap.Logger) *EmailVerificationGuard {
	return &EmailVerificationGuard{
		logger: logger,
	}
}

func (this *EmailVerificationGuard) Act(ctx fiber.Ctx) error {

	app, ok := ctx.Locals("app").(*entity.App)

	if !ok || app == nil {
		return e.ThrowInternalServerError("App context missing in EmailVerificationGuard - should be run after AppGuard")
	}

	user, ok := ctx.Locals("user").(*entity.User)
	if !ok || user == nil {
		return e.ThrowInternalServerError("User context missing in EmailVerificationGuard - should be run after AuthGuard")
	}

	if app.VerifyEmail && !user.VerifyEmail {
		return e.ThrowUnverifiedEmail("App requires email to be verified ti authenticate", utils.JSON{
			"email": user.Email,
		})
	}

	return ctx.Next()
}
