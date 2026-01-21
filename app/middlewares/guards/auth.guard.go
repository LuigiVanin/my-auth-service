package guards

import (
	"auth_service/app/modules/authorize/services"
	e "auth_service/common/errors"
	"auth_service/common/global"
	entity "auth_service/infra/entities"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AuthGuard struct {
	authService services.IAuthorizeService
}

func NewAuthGuard(authService services.IAuthorizeService) *AuthGuard {
	return &AuthGuard{
		authService: authService,
	}
}

func (guard *AuthGuard) Act(ctx *fiber.Ctx) error {
	global.Logger.Info("Auth Guard Triggered")

	app, ok := ctx.Locals("app").(*entity.App)
	if !ok || app == nil {
		// AppGuard must be run before AuthGuard
		return e.ThrowInternalServerError("App context missing in AuthGuard")
	}

	authorization := ctx.Get("Authorization")

	if authorization == "" {
		return e.ThrowUnauthorizedError("Missing Authorization header")
	}

	// Validate the token using AuthorizeService
	// We pass ctx.IP() to validate IP address mismatch as done in AuthorizeController
	res, err := guard.authService.Authorize(app, authorization, ctx.IP())

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
