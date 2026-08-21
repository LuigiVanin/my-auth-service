package guards

import (
	e "auth_service/app/errors"
	"auth_service/app/modules/authorize/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/global"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AuthGuard struct {
	authService services.IAuthorizeService
	logger      *zap.Logger
}

func NewAuthGuard(authService services.IAuthorizeService, logger *zap.Logger) *AuthGuard {
	return &AuthGuard{
		authService: authService,
		logger:      logger,
	}
}

func (this *AuthGuard) Act(ctx *fiber.Ctx) error {
	this.logger.Info("Auth Guard Triggered")

	app, ok := ctx.Locals("app").(*entity.App)
	if !ok || app == nil {
		return e.ThrowInternalServerError("App context missing in AuthGuard - should be run before AuthGuard")
	}

	authorization := ctx.Get("Authorization")

	if authorization == "" {
		return e.ThrowUnauthorizedError("Missing Authorization header")
	}

	// Validate the token using AuthorizeService
	// We pass ctx.IP() to validate IP address mismatch as done in AuthorizeController
	res, err := this.authService.Authorize(app, authorization, ctx.IP())

	if err != nil {
		return err
	}

	if !res.Authorized {
		return e.ThrowUnauthorizedError("Unauthorized")
	}

	global.Logger.Info(
		"User authorized",
		zap.Uint("user_id", res.User.ID),
		zap.String("session_id", res.SessionId),
	)

	// Store user and session info in context for downstream handlers
	ctx.Locals("user", &res.User)
	ctx.Locals("session_id", res.SessionId)
	ctx.Locals("token_type", res.TokenType)

	return ctx.Next()
}
