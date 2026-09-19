package authorize

import (
	"auth_service/app/modules/authorize/controller"
	"auth_service/app/modules/authorize/services"
	sr "auth_service/app/modules/core/session/repository"
	ur "auth_service/app/modules/core/user/repository"
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

	// Held for the writes inside the transaction of ChangeEmail only - the reads
	// keep going through IUserService.
	fx.Provide(
		fx.Private,
		fx.Annotate(
			ur.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
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
