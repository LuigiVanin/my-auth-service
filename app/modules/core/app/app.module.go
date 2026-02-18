package app

import (
	"auth_service/app/modules/core/app/controller"
	ar "auth_service/app/modules/core/app/repository"
	"auth_service/app/modules/core/app/services"

	"github.com/gofiber/fiber/v2"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"app",

	fx.Provide(fx.Annotate(
		ar.NewAppRepository,
		fx.As(new(ar.IAppRepository)),
	)),

	fx.Provide(
		fx.Private,
		fx.Annotate(
			services.NewAppService,
			fx.As(new(services.IAppService)),
		),
	),

	fx.Provide(
		fx.Private,
		controller.NewAppController,
	),

	fx.Invoke(func(server *fiber.App, controller *controller.AppController) {
		controller.Register(server)
	}),
)
