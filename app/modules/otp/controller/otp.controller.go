package controller

import (
	"auth_service/app/modules/otp/services"
	"auth_service/common/constants"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"

	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &OtpController{}

type OtpController struct {
	otpService services.IOtpService
}

func NewOtpController(otpService services.IOtpService) *OtpController {
	return &OtpController{
		otpService: otpService,
	}
}

type GenerateConsumableDto struct {
	Action constants.AuthAction `json:"action"`
}

func (c *OtpController) GenerateConsumable(ctx *fiber.Ctx) error {
	var body GenerateConsumableDto
	if err := ctx.BodyParser(&body); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	res, err := c.otpService.GenerateConsumable(body.Action)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (c *OtpController) Register(server *fiber.App) {
	group := server.Group("/otp")
	group.Post("/generate_consumable", c.GenerateConsumable)
	group.Post("/verify_email")

}
