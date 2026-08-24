package participant

import (
	pr "auth_service/app/modules/core/participant/repository"
	"auth_service/app/modules/core/participant/services"

	"go.uber.org/fx"
)

// No controller: a participation is always read through its organization, so the
// routes live on OrganizationController.
var Module = fx.Module(
	"participant",

	fx.Provide(
		fx.Annotate(
			pr.NewParticipantRepository,
			fx.As(new(pr.IParticipantRepository)),
		),
	),

	fx.Provide(
		fx.Annotate(
			services.NewParticipantService,
			fx.As(new(services.IParticipantService)),
		),
	),
)
