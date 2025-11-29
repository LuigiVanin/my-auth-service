package interfaces

import "github.com/gofiber/fiber/v2"

type IGuard interface {
	Act(ctx *fiber.Ctx) error
}
