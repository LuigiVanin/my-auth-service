package invite

import (
	"auth_service/app/modules/core/invite/controller"
	"auth_service/app/modules/core/invite/repository"
	"auth_service/app/modules/core/invite/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"invite",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			repository.NewInviteRepository,
			fx.As(new(repository.IInviteRepository)),
		),
	),

	// Exported: register consumes it for the signup by invite.
	fx.Provide(
		fx.Annotate(
			services.NewInviteService,
			fx.As(new(services.IInviteService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewInviteController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
