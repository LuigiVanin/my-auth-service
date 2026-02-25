package user

import (
	"auth_service/app/modules/core/user/controller"
	ur "auth_service/app/modules/core/user/repository"
	"auth_service/app/modules/core/user/services"

	"github.com/gofiber/fiber/v2"
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
		fx.Private,
		controller.NewUserController,
	),

	fx.Invoke(func(server *fiber.App, controller *controller.UserController) {
		controller.Register(server)
	}),
)
