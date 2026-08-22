package interfaces

import "github.com/gofiber/fiber/v3"

type IGuard interface {
	Act(ctx fiber.Ctx) error
}
