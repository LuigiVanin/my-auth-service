package user_pool

import (
	"auth_service/app/modules/user_pool/repository"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"user_pool",

	fx.Provide(fx.Annotate(
		repository.NewUserPoolRepository,
		fx.As(new(repository.IUserPoolRepository)),
	)),
)
