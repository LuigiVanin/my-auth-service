package global

import (
	ur "auth_service/app/modules/user/repository"

	"go.uber.org/fx"
)

// TODO: add this to main and remove the usage of user repository in the individual modules
var Module = fx.Module(
	"global",

	fx.Provide(
		fx.Annotate(
			ur.NewUserRepository,
			fx.As(new(ur.IUserRepository)),
		),
	),
)
