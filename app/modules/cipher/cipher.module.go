package cipher

import (
	"auth_service/app/modules/cipher/services"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"cipher",

	fx.Provide(
		fx.Annotate(
			services.NewCipherService,
			fx.As(new(services.ICipherService)),
		),
	),
)
