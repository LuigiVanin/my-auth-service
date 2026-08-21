package register

import (
	"auth_service/app/modules/core/user/repository"
	"auth_service/app/modules/register/controller"
	"auth_service/app/modules/register/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"register",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			repository.NewUserRepository,
			fx.As(new(repository.IUserRepository)),
		),
	),

	fx.Provide(
		fx.Private,
		fx.Annotate(
			services.NewRegisterService,
			fx.As(new(services.IRegisterService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewRegisterController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
