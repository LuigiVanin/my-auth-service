package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/organization/models"
	services "auth_service/app/modules/core/organization/services"
	pdto "auth_service/app/modules/core/participant/models"
	ps "auth_service/app/modules/core/participant/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type OrganizationController struct {
	organizationService services.IOrganizationService
	participantService  ps.IParticipantService

	authGuard         *guards.AuthGuard
	organizationGuard *guards.OrganizationGuard
	permissionsGuard  *guards.PermissionsGuard

	logger  *zap.Logger
	swagger *openapi.Builder
}

var _ interfaces.IController = &OrganizationController{}

func NewOrganizationController(
	organizationService services.IOrganizationService,
	participantService ps.IParticipantService,
	authGuard *guards.AuthGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
) *OrganizationController {
	return &OrganizationController{
		organizationService: organizationService,
		participantService:  participantService,
		authGuard:           authGuard,
		organizationGuard:   organizationGuard,
		permissionsGuard:    permissionsGuard,
		logger:              logger,
		swagger:             builder,
	}
}

func (this *OrganizationController) GetOrganizations(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)

	var query dto.GetOrganizationsQuery

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	organizations, err := this.organizationService.FindAllForUser(currentUser.ID, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(organizations)
}

func (this *OrganizationController) CreateOrganization(ctx fiber.Ctx) error {
	var payload dto.CreateOrganizationPayload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	organization, err := this.organizationService.CreateForUser(currentUser, currentOrganization, &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(organization)
}

func (this *OrganizationController) SwitchOrganization(ctx fiber.Ctx) error {
	var payload dto.SwitchOrganizationPayload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentUser := ctx.Locals("user").(*entity.User)

	organization, err := this.organizationService.Switch(currentUser, payload.OrganizationId)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(organization)
}

func (this *OrganizationController) GetParticipants(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)

	organizationId := ctx.Params("id")

	if organizationId == "" {
		return e.ThrowBadRequest("Organization id is required")
	}

	var query pdto.GetParticipantsQuery

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	participants, err := this.participantService.FindAllInOrganization(organizationId, currentUser.ID, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(participants)
}

func (this *OrganizationController) Register(server *fiber.App) {

	group := server.Group("/core")

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/organizations", openapi.Options{
			Summary:     "List organizations",
			Description: "Lists the organizations the authenticated user participates in, whatever the organization it is currently scoped to.",
			Tags:        []string{docs.TagOrganizations},
		}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Organizations to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Organizations per page - defaults to 10",
			}).
			AddResponse(fiber.StatusOK, dto.GetOrganizationsResponse{}, openapi.Options{
				Description: "Page of organizations, each with the profile that is its permission ceiling",
			}),
	)
	group.Get(
		"/organizations",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetOrganizations,
	)

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/organizations", openapi.Options{
				Summary:     "Create an organization",
				Description: "Creates an organization in the pool of the authenticated user, who becomes its owner and its first participant. `profile_id` is the permission ceiling of the new organization and can never grant more than the organization the request is made from; when it is absent the default profile of the pool is used. The current organization of the user is not changed - use `PUT /core/organizations/switch` for that.",
				Tags:        []string{docs.TagOrganizations},
			}),
		).
			AddBody(dto.CreateOrganizationPayload{}, openapi.Options{
				Required:    true,
				Description: "Name, description and optional permission ceiling of the organization",
			}).
			AddResponse(fiber.StatusCreated, entity.Organization{}, openapi.Options{
				Description: "The created organization",
			}),
	)
	group.Post(
		"/organizations",
		middleware.BodyValidator[dto.CreateOrganizationPayload](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.CreateOrganization,
	)

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "PUT", "/core/organizations/switch", openapi.Options{
				Summary:     "Switch the current organization",
				Description: "Moves the scope of the authenticated user to another organization it participates in. Every listing and mutation is filtered by this scope, so this is how a user reaches the resources of another of its organizations.",
				Tags:        []string{docs.TagOrganizations},
			}),
		).
			AddBody(dto.SwitchOrganizationPayload{}, openapi.Options{
				Required:    true,
				Description: "Id of the organization to switch to",
			}).
			AddResponse(fiber.StatusOK, entity.Organization{}, openapi.Options{
				Description: "The organization the user is now scoped to",
			}),
	)
	group.Put(
		"/organizations/switch",
		middleware.BodyValidator[dto.SwitchOrganizationPayload](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.SwitchOrganization,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/organizations/{id}/participants", openapi.Options{
			Summary:     "List the participants of an organization",
			Description: "Lists who takes part in the organization and with which profile. The authenticated user has to participate in the organization itself.",
			Tags:        []string{docs.TagOrganizations},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the organization",
			}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Participants to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Participants per page - defaults to 10",
			}).
			AddResponse(fiber.StatusOK, pdto.GetParticipantsResponse{}, openapi.Options{
				Description: "Page of participants, each with its user and its profile",
			}),
	)
	group.Get(
		"/organizations/:id/participants",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetParticipants,
	)
}
