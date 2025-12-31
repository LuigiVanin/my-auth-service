package login

import (
	"auth_service/app/modules/login/controller"
	"auth_service/app/modules/login/services"
	ts "auth_service/app/modules/tokenizer/services"
	ur "auth_service/app/modules/user/repository"

	"github.com/gofiber/fiber/v2"
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
		fx.Private,
		fx.Annotate(
			ts.NewTokenizerService,
			fx.As(new(ts.ITokenizerService)),
		),
	),

	fx.Provide(controller.NewLoginController),

	fx.Invoke(func(server *fiber.App, controller *controller.LoginController) {
		controller.Register(server)
	}),
)
