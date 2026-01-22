package profile

import (
	"auth_service/app/modules/core/profile/repository"
	"auth_service/app/modules/core/profile/services"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			repository.NewProfileRepository,
			fx.As(new(repository.IProfileRepository)),
		),
		fx.Annotate(
			services.NewProfileService,
			fx.As(new(services.IProfileService)),
		),
	),
)
