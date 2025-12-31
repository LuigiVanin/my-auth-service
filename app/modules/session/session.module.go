package session

import (
	"auth_service/app/modules/session/repository"
	"auth_service/app/modules/session/services"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"session",
	fx.Provide(
		fx.Private,
		fx.Annotate(
			repository.NewSessionRepository,
			fx.As(new(repository.ISessionRepository)),
		),
	),

	fx.Provide(
		fx.Annotate(
			services.NewSessionService,
			fx.As(new(services.ISessionService)),
		),
	),
)
