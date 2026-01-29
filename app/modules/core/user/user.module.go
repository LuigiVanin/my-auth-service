package user

import (
	"auth_service/app/modules/core/user/repository"
	ur "auth_service/app/modules/core/user/repository"
	"auth_service/app/modules/core/user/services"

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
		fx.Private,
		fx.Annotate(
			repository.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
		)),
)
