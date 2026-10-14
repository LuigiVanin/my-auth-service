package guards

import (
	e "auth_service/app/errors"
	ps "auth_service/app/modules/core/participant/services"
	entity "auth_service/infra/entities"
	i "auth_service/shared/interfaces"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

var _ i.IGuard = &OrganizationGuard{}

// OrganizationGuard resolves the scope of the request: the organization the user
// currently sits in, and the participation that gives it its powers there.
//
// It must be chained per route, after AuthGuard. Mounted by prefix in routes.go it
// would run before the route level AuthGuard, so `user` would not be in Locals.
type OrganizationGuard struct {
	participantService ps.IParticipantService
	logger             *zap.Logger
}

func NewOrganizationGuard(participantService ps.IParticipantService, logger *zap.Logger) *OrganizationGuard {
	return &OrganizationGuard{
		participantService: participantService,
		logger:             logger,
	}
}

func (this *OrganizationGuard) Act(ctx fiber.Ctx) error {
	if _, err := readLocal[entity.App](ctx, "app", "OrganizationGuard", "AppGuard"); err != nil {
		return err
	}

	user, err := readLocal[entity.User](ctx, "user", "OrganizationGuard", "AuthGuard")

	if err != nil {
		return err
	}

	// AuthorizeService preloads it, and current_organization_id is NOT NULL, so a
	// nil here is a broken row rather than a caller mistake.
	organization := user.CurrentOrganization

	if organization == nil {
		this.logger.Error(
			"User has no current organization loaded",
			zap.Uint("user_id", user.ID),
			zap.String("current_organization_id", user.CurrentOrganizationId),
		)

		return e.ThrowInternalServerError("The current organization of the user could not be resolved")
	}

	if organization.Profile == nil {
		return e.ThrowInternalServerError("The current organization of the user has no profile loaded")
	}

	participant, err := this.participantService.FindForUserInOrganization(organization.ID, user.ID)

	if err != nil {
		return err
	}

	if participant == nil {
		this.logger.Warn(
			"User is not a participant of its current organization",
			zap.Uint("user_id", user.ID),
			zap.String("organization_id", organization.ID),
		)

		return e.ThrowNotAParticipant("User does not participate in its current organization")
	}

	if participant.Profile == nil {
		return e.ThrowInternalServerError("The participation of the user has no profile loaded")
	}

	ctx.Locals("organization", organization)
	ctx.Locals("participant", participant)

	return ctx.Next()
}
