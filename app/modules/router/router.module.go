package router

import (
	"auth_service/app/middlewares/guards"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"router",
	fx.Invoke(RegisterMiddlewares),
)

func RegisterMiddlewares(server *fiber.App, appGuard *guards.AppGuard) {
	server.Group("/auth", appGuard.Act)
}
