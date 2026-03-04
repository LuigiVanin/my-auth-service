package user_pool

import (
	"auth_service/app/modules/core/user_pool/controller"
	"auth_service/app/modules/core/user_pool/repository"

	"github.com/gofiber/fiber/v2"
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
		fx.Private,
		controller.NewUserPoolController,
	),

	fx.Invoke(func(controller *controller.UserPoolController, server *fiber.App) {
		controller.Register(server)
	}),
)
