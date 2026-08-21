package login

import (
	ur "auth_service/app/modules/core/user/repository"
	"auth_service/app/modules/login/controller"
	"auth_service/app/modules/login/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"login",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			ur.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
		),
	),

	fx.Provide(
		fx.Private,
		fx.Annotate(
			services.NewLoginService,
			fx.As(new(services.ILoginService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewLoginController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
