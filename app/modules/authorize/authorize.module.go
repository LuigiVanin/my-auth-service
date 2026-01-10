package authorize

import (
	"auth_service/app/modules/authorize/controller"
	"auth_service/app/modules/authorize/services"
	sr "auth_service/app/modules/session/repository"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"authorize",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			sr.NewSessionRepository,
			fx.As(new(sr.ISessionRepository)),
		),
	),

	fx.Provide(fx.Annotate(
		services.NewAuthorizeService,
		fx.As(new(services.IAuthorizeService)),
	)),

	fx.Provide(controller.NewAuthorizeController),

	fx.Invoke(func(server *fiber.App, controller *controller.AuthorizeController) {
		controller.Register(server)
	}),
)
