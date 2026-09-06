package organization

import (
	"auth_service/app/modules/core/organization/controller"
	or "auth_service/app/modules/core/organization/repository"
	"auth_service/app/modules/core/organization/services"
	"auth_service/shared/interfaces"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"organization",

	fx.Provide(
		fx.Annotate(
			or.NewOrganizationRepository,
			fx.As(new(or.IOrganizationRepository)),
		),
	),

	fx.Provide(
		fx.Annotate(
			services.NewOrganizationService,
			fx.As(new(services.IOrganizationService)),
		),
	),

	fx.Provide(
		fx.Annotate(
			controller.NewOrganizationController,
			fx.As(new(interfaces.IController)),
			fx.ResultTags(`group:"controllers"`),
		),
	),
)
