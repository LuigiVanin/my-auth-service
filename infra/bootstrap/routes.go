package bootstrap

import (
	"auth_service/app/middlewares/guards"

	"github.com/gofiber/fiber/v2"
)

func RegisterGuards(server *fiber.App, appGuard *guards.AppGuard) {
	server.Use("/auth", appGuard.Act)
	server.Use("/otp", appGuard.Act)
}
