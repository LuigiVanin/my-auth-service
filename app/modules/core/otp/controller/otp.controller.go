package controller

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"

	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &OtpController{}

type OtpController struct {
	otpService services.IOtpService
	otpGuard   *guards.OtpGuard
	appGuard   *guards.AppGuard
}

func NewOtpController(otpService services.IOtpService, otpGuard *guards.OtpGuard, appGuard *guards.AppGuard) *OtpController {
	return &OtpController{
		otpService: otpService,
		otpGuard:   otpGuard,
		appGuard:   appGuard,
	}
}

func (c *OtpController) GenerateConsumable(ctx *fiber.Ctx) error {

	app := ctx.Locals("app").(*entity.App)

	var body dto.ConsumableOtpPayload
	if err := ctx.BodyParser(&body); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	body.Metadata["Ip"] = ctx.IP()

	res, err := c.otpService.GenerateConsumable(app, body.Action, body.Metadata)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *OtpController) Register(server *fiber.App) {
	group := server.Group("/otp")

	group.Post(
		"/generate_consumable",
		middleware.BodyValidator[dto.ConsumableOtpPayload](),
		validators.OtpValidator,
		this.otpGuard.Act,
		this.GenerateConsumable,
	)

}
