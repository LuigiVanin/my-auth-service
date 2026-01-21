package otp

import (
	"auth_service/app/modules/otp/controller"
	"auth_service/app/modules/otp/repository"
	"auth_service/app/modules/otp/services"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"otp",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			repository.NewOtpRepository,
			fx.As(new(repository.IOtpRepository)),
		),
	),

	fx.Provide(
		fx.Private,
		fx.Annotate(
			services.NewOtpService,
			fx.As(new(services.IOtpService)),
		),
	),

	fx.Provide(controller.NewOtpController),

	fx.Invoke(func(controller *controller.OtpController, server *fiber.App) {
		controller.Register(server)
	}),
)
