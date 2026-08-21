package user_pool

import (
	"auth_service/app/modules/core/user_pool/controller"
	"auth_service/app/modules/core/user_pool/repository"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"user_pool",

	fx.Provide(fx.Annotate(
		repository.NewUserPoolRepository,
		fx.As(new(repository.IUserPoolRepository)),
	)),

	// fx.Provide(fx.Annotate(
	// 	repository.NewUserPoolService,
	// 	fx.As(new(repository.IUserPoolService)),
	// )),

	fx.Provide(
		fx.Annotate(
			controller.NewUserPoolController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
