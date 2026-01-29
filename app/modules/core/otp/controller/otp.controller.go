package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/services"
	"auth_service/common/constants"
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

func (this *OtpController) GenerateConsumable(ctx *fiber.Ctx) error {

	app := ctx.Locals("app").(*entity.App)

	// Extract action from query params
	actionStr := ctx.Query("action")
	if actionStr == "" {
		return e.ThrowBadRequest("Action query parameter is required")
	}

	// Parse the entire body as the payload
	var bodyPayload map[string]any
	if err := ctx.BodyParser(&bodyPayload); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	// Build the ConsumableOtpPayload
	payload := dto.ConsumableOtpPayload{
		Action:  constants.AuthAction(actionStr),
		Contact: "", // Will be extracted from payload in service
		Payload: bodyPayload,
	}

	// Get IP address from context
	ip := ctx.IP()

	res, err := this.otpService.GenerateConsumable(app, payload, ip)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *OtpController) Register(server *fiber.App) {
	group := server.Group("/otp")

	group.Post(
		"/generate_consumable",
		validators.OtpValidator,
		this.otpGuard.Act,
		this.GenerateConsumable,
	)

}
