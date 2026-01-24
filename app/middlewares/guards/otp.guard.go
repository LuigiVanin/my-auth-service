package guards

import (
	"auth_service/app/models/dto"
	"auth_service/common/constants"
	e "auth_service/common/errors"
	i "auth_service/common/interfaces"
	entity "auth_service/infra/entities"

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

	var body dto.ConsumableOtpPayload
	if err := ctx.BodyParser(&body); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	if body.Action == constants.ActionRegister {
		return ctx.Next()
	}

	err := this.authGuard.Act(ctx)

	if err != nil {
		return err
	}

	return ctx.Next()
}
