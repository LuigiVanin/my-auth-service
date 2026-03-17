package healthcheck

import (
	"auth_service/common/interfaces"

	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &HealthCheckController{}

type HealthCheckController struct {
}

func NewHealthCheckController() *HealthCheckController {
	return &HealthCheckController{}
}

func (this *HealthCheckController) Status(ctx *fiber.Ctx) error {

	return ctx.JSON(fiber.Map{
		"status": "ok",
	})
}

func (this *HealthCheckController) Register(ctx *fiber.App) {
	ctx.Get("/health-check", this.Status)
}
