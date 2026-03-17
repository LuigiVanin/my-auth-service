package healthcheck

import (
	"auth_service/common/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"healthcheck",

	fx.Provide(
		fx.Annotate(
			NewHealthCheckController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
