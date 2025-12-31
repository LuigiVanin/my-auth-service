package hash

import (
	"auth_service/app/modules/hash/services"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"hash",

	fx.Provide(
		fx.Annotate(
			services.NewHashService,
			fx.As(new(services.IHashService)),
		),
	),
)
