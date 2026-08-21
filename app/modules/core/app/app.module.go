package app

import (
	"auth_service/app/modules/core/app/controller"
	ar "auth_service/app/modules/core/app/repository"
	"auth_service/app/modules/core/app/services"
	"auth_service/shared/interfaces"

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
		fx.Annotate(
			controller.NewAppController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
