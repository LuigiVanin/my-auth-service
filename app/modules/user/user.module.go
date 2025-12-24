package user

import (
	"auth_service/app/modules/user/repository"
	ur "auth_service/app/modules/user/repository"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"user",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			repository.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
		)),
)
