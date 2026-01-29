package validators

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	"auth_service/common/constants"
	e "auth_service/common/errors"

	"github.com/gofiber/fiber/v2"
)

func OtpValidator(ctx *fiber.Ctx) error {
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
	default:
		return e.ThrowUnprocessableEntity("Invalid OTP action")
	}
}
