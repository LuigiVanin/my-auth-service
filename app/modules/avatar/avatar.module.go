package avatar

import (
	"auth_service/app/modules/avatar/controller"
	"auth_service/app/modules/avatar/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"avatar",

	fx.Provide(
		fx.Private,
		fx.Annotate(
			services.NewAvatarService,
			fx.As(new(services.IAvatarService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewAvatarController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
