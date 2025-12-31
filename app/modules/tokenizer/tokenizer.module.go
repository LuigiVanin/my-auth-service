package tokenizer

import (
	"auth_service/app/modules/tokenizer/services"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"tokenizer",

	fx.Provide(
		fx.Annotate(
			services.NewTokenizerService,
			fx.As(new(services.ITokenizerService)),
		),
	),
)
