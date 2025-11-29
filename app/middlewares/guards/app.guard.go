package guards

import (
	e "auth_service/common/errors"

	"github.com/gofiber/fiber/v2"
)

type AppGuard struct {
}

func NewAppGuard() *AppGuard {
	return &AppGuard{}
}

func (guard *AppGuard) Act(ctx *fiber.Ctx) error {
	appKey := ctx.Get("X-App-Key")
	poolKey := ctx.Get("X-Pool-Key")

	if appKey == "" || poolKey == "" {
		return e.ThrowUnauthorizedError("X-App-Key and X-Pool-Key are required")
	}

	return ctx.Next()
}
