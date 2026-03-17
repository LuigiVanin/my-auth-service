package otp

import (
	"auth_service/app/modules/core/otp/controller"
	"auth_service/app/modules/core/otp/repository"
	"auth_service/app/modules/core/otp/services"
	"auth_service/common/interfaces"

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
		fx.Annotate(
			services.NewOtpService,
			fx.As(new(services.IOtpService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewOtpController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
