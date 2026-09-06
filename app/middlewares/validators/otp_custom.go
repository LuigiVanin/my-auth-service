package validators

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/core/otp/models"
	"auth_service/shared/constants"

	"github.com/gofiber/fiber/v3"
)

func OtpValidator(ctx fiber.Ctx) error {
	// Extract action from query params
	action := ctx.Query("action")
	if action == "" {
		return e.ThrowBadRequest("Action query parameter is required")
	}

	// Validate action and apply appropriate body validator
	switch constants.AuthAction(action) {
	case constants.ActionRegister:
		return middleware.BodyValidator[dto.OtpRegisterPayload]()(ctx)
	case constants.ActionLogin:
		return middleware.BodyValidator[dto.OtpLoginPayload]()(ctx)
	case constants.ActionForgotPassword:
		return middleware.BodyValidator[dto.OtpForgotPasswordPayload]()(ctx)
	default:
		return e.ThrowUnprocessableEntity("Invalid OTP action")
	}
}
