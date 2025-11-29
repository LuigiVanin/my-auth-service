package validators

import (
	middleware "auth_service/app/middlewares"
	"auth_service/app/models/dto"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func RegisterValidator(ctx *fiber.Ctx) error {
	registerMethod := ctx.Queries()["method"]

	fmt.Println(registerMethod)

	return (middleware.BodyValidator[dto.RegisterRequestWithPassoword]())(ctx)
}
