package validators

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/login/models"

	"github.com/gofiber/fiber/v3"
)

func LoginValidator(ctx fiber.Ctx) error {
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
