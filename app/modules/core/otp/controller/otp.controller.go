package controller

import (
	"auth_service/app/apidocs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &OtpController{}

type OtpController struct {
	otpService services.IOtpService
	otpGuard   *guards.OtpGuard
	appGuard   *guards.AppGuard

	swagger *openapi.Builder
}

func NewOtpController(
	otpService services.IOtpService,
	otpGuard *guards.OtpGuard,
	appGuard *guards.AppGuard,
	builder *openapi.Builder,
) *OtpController {
	return &OtpController{
		otpService: otpService,
		otpGuard:   otpGuard,
		appGuard:   appGuard,
		swagger:    builder,
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
		Contact: "", // Will be extracted from payload in service
		Payload: bodyPayload,
	}

	// Get IP address from context
	ip := ctx.IP()

	res, err := this.otpService.GenerateConsumable(
		constants.AuthAction(actionStr),
		app,
		payload,
		ip,
	)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *OtpController) VerifyConsumable(ctx *fiber.Ctx) error {
	otpId := ctx.Params("otp_id")
	if otpId == "" {
		return e.ThrowBadRequest("OTP id is required")
	}

	var payload dto.VerifyConsumableOtpPayload
	if err := ctx.BodyParser(&payload); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	app := ctx.Locals("app").(*entity.App)

	res, err := this.otpService.VerifyConsumable(otpId, payload.Code, app.ID)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *OtpController) Register(server *fiber.App) {
	group := server.Group("/otp")

	// The body of the route depends on `action`: the documented schema is the
	// one of `REGISTER`, `LOGIN` only takes the `email` field.
	this.swagger.Add(
		apidocs.Validated(
			apidocs.AppRoute(this.swagger, "POST", "/otp/generate_consumable", openapi.Options{
				Summary:     "Generate an OTP",
				Description: "Generates a one time password for an action and sends it to the contact of the payload. The code is then consumed by the endpoint owning the action - /auth/register, /auth/login or /auth/forgot_password.",
				Tags:        []string{apidocs.TagOtp},
			}),
		).
			AddQueryParam("action", openapi.String, openapi.Options{
				Required:    true,
				Description: "Action the OTP is generated for: `REGISTER`, `LOGIN`, `VERIFY_EMAIL`, `TWO_FA`, `FORGOT_PASSWORD`, `CHANGE_EMAIL` or `REGEN_APP_SECRET_KEY`",
			}).
			AddHeaderParam("Authorization", openapi.String, openapi.Options{
				Description: "Access token - required for every action other than `REGISTER`, `LOGIN` and `FORGOT_PASSWORD`",
			}).
			AddBody(dto.OtpRegisterPayload{}, openapi.Options{
				Required:    true,
				Description: "Contact the code is sent to - shape depends on the `action` query parameter",
			}).
			AddResponse(fiber.StatusOK, dto.GenerateConsumableOtpResponse{}, openapi.Options{
				Description: "OTP generated - the returned `otp_id` is what the consuming endpoint expects",
			}),
	)

	group.Post(
		"/generate_consumable",
		validators.OtpValidator,
		this.otpGuard.Act,
		this.GenerateConsumable,
	)

	this.swagger.Add(
		apidocs.Validated(
			apidocs.AppRoute(this.swagger, "PUT", "/otp/verify/{otp_id}", openapi.Options{
				Summary:     "Verify an OTP",
				Description: "Checks the code of a generated OTP and returns the metadata stored with it. The OTP is only consumed later, by the endpoint owning its action.",
				Tags:        []string{apidocs.TagOtp},
			}),
		).
			AddPathParam("otp_id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id returned by /otp/generate_consumable",
			}).
			AddBody(dto.VerifyConsumableOtpPayload{}, openapi.Options{
				Required:    true,
				Description: "The code received by the contact",
			}).
			AddResponse(fiber.StatusOK, services.VerifyConsumableOtpResponse{}, openapi.Options{
				Description: "Result of the verification and the metadata stored with the OTP",
			}),
	)

	group.Put(
		"/verify/:otp_id",
		middleware.BodyValidator[dto.VerifyConsumableOtpPayload](),
		this.VerifyConsumable,
	)

}
