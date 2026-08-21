package authorize

import (
	"auth_service/app/modules/authorize/controller"
	"auth_service/app/modules/authorize/services"
	sr "auth_service/app/modules/core/session/repository"
	"auth_service/shared/interfaces"

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

	fx.Provide(
		fx.Annotate(
			controller.NewAuthorizeController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
