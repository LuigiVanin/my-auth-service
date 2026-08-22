package middleware

import "github.com/gofiber/fiber/v3"

func Json(ctx fiber.Ctx) error {
	ctx.Accepts("application/json")

	return ctx.Next()
}
