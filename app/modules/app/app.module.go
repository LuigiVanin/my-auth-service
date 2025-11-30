package app

import (
	ar "auth_service/app/modules/app/repository"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"app",

	fx.Provide(fx.Annotate(
		ar.NewAppRepository,
		fx.As(new(ar.IAppRepository)),
	)),
)
