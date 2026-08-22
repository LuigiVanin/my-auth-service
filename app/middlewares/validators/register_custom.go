package validators

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/register/models"

	"github.com/gofiber/fiber/v3"
)

func RegisterValidator(ctx fiber.Ctx) error {
	registerMethod := ctx.Queries()["method"]
	validatorMiddleware := middleware.BodyValidator[dto.RegisterPayloadWithPassoword]()

	switch registerMethod {
	case "otp", "":
		validatorMiddleware = middleware.BodyValidator[dto.RegisterPayloadWithOtp]()
	case "password":
		validatorMiddleware = middleware.BodyValidator[dto.RegisterPayloadWithPassoword]()
	default:
		return e.ThrowUnprocessableEntity("Invalid register method")
	}

	return validatorMiddleware(ctx)
}
