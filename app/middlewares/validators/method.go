package validators

import (
	middleware "auth_service/app/middlewares"
	e "auth_service/common/errors"

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
