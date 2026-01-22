package jwt

import "go.uber.org/fx"

var Module = fx.Module(
	"jwt",

	fx.Provide(
		fx.Annotate(
			NewJwtService,
			fx.As(new(IJwtService)),
		),
	),
)
