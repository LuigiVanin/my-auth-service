package validators

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	e "auth_service/common/errors"

	"github.com/gofiber/fiber/v2"
)

func RegisterValidator(ctx *fiber.Ctx) error {
	registerMethod := ctx.Queries()["method"]
	validatorMiddleware := middleware.BodyValidator[dto.RegisterRequestWithPassoword]()

	switch registerMethod {
	case "password":
	case "":
		validatorMiddleware = middleware.BodyValidator[dto.RegisterRequestWithPassoword]()
	case "otp":
		validatorMiddleware = middleware.BodyValidator[dto.RegisterRequestWithOtp]()
	default:
		return e.ThrowUnprocessableEntity("Invalid register method")
	}

	return validatorMiddleware(ctx)
}
