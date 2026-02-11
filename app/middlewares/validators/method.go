package validators

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"

	"github.com/gofiber/fiber/v2"
)

func MethodValidator[PasswordValidator any, OtpValidator any]() func(ctx *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		loginMethod := ctx.Queries()["method"]
		validatorMiddleware := middleware.BodyValidator[PasswordValidator]()

		switch loginMethod {
		case "password":
		case "":
			validatorMiddleware = middleware.BodyValidator[PasswordValidator]()
		case "otp":
			validatorMiddleware = middleware.BodyValidator[OtpValidator]()
		default:
			return e.ThrowUnprocessableEntity("Invalid login method")
		}

		return validatorMiddleware(ctx)
	}
}
