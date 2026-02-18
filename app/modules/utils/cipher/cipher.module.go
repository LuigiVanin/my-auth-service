package cipher

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"cipher",

	fx.Provide(
		fx.Annotate(
			NewCipherService,
			fx.As(new(ICipherService)),
		),
	),
)
