package user

import (
	"auth_service/app/modules/core/user/controller"
	ur "auth_service/app/modules/core/user/repository"
	"auth_service/app/modules/core/user/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"user",

	fx.Provide(
		fx.Annotate(
			services.NewUserService,
			fx.As(new(services.IUserService)),
		)),

	fx.Provide(
		fx.Annotate(
			ur.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
		)),

	fx.Provide(
		fx.Annotate(
			controller.NewUserController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
