package controller

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"
	"fmt"

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

func (c *OtpController) GenerateConsumable(ctx *fiber.Ctx) error {
	var body dto.ConsumableOtpPayload

	app := ctx.Locals("app").(*entity.App)

	fmt.Println("APP: ", app)
	fmt.Println("BODY: ", body)

	if err := ctx.BodyParser(&body); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	res, err := c.otpService.GenerateConsumable(app, body.Action, body.Metadata)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (c *OtpController) Register(server *fiber.App) {
	group := server.Group("/otp")
	group.Post(
		"/generate_consumable",
		middleware.BodyValidator[dto.ConsumableOtpPayload](),
		c.GenerateConsumable,
	)
	// group.Post("/verify_email")

}
