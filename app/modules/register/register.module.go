package register

import (
	"auth_service/app/modules/register/controller"
	"auth_service/app/modules/register/services"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"register",

	fx.Provide(
		fx.Annotate(
			services.NewRegisterService,
			fx.As(new(services.IRegisterService)),
		),
	),

	fx.Provide(controller.NewRegisterController),

	fx.Invoke(func(server *fiber.App, controller *controller.RegisterController) {
		controller.Register(server)
	}),
)
