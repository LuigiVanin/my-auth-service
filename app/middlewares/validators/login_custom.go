package validators

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	e "auth_service/common/errors"

	"github.com/gofiber/fiber/v2"
)

func LoginValidator(ctx *fiber.Ctx) error {
	loginMethod := ctx.Queries()["method"]
	validatorMiddleware := middleware.BodyValidator[dto.LoginPayloadWithPassoword]()

	switch loginMethod {
	case "password":
	case "":
		validatorMiddleware = middleware.BodyValidator[dto.LoginPayloadWithPassoword]()
	case "otp":
		validatorMiddleware = middleware.BodyValidator[dto.LoginPayloadWithOtp]()
	default:
		return e.ThrowUnprocessableEntity("Invalid login method")
	}

	return validatorMiddleware(ctx)
}
