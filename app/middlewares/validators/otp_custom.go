package validators

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	"auth_service/common/constants"
	e "auth_service/common/errors"

	"github.com/gofiber/fiber/v2"
)

func OtpValidator(ctx *fiber.Ctx) error {
	var payload dto.ConsumableOtpPayload

	if err := ctx.BodyParser(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	switch payload.Action {
	case constants.ActionRegister:
		return middleware.BodyValidator[dto.OtpRegisterPayload]()(ctx)
	case constants.ActionLogin:
		return middleware.BodyValidator[dto.OtpLoginActionPayload]()(ctx)
	default:
		return e.ThrowUnprocessableEntity("Invalid OTP action")
	}
}
