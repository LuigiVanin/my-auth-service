package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/profile/models"
	services "auth_service/app/modules/core/profile/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"
	"auth_service/shared/permissions"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

type ProfileController struct {
	profileService services.IProfileService

	authGuard         *guards.AuthGuard
	organizationGuard *guards.OrganizationGuard
	permissionsGuard  *guards.PermissionsGuard

	swagger *openapi.Builder
}

var _ interfaces.IController = &ProfileController{}

func NewProfileController(
	profileService services.IProfileService,
	authGuard *guards.AuthGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	builder *openapi.Builder,
) *ProfileController {
	return &ProfileController{
		profileService:    profileService,
		authGuard:         authGuard,
		organizationGuard: organizationGuard,
		permissionsGuard:  permissionsGuard,
		swagger:           builder,
	}
}

type getGrantsResponse struct {
	Data      []permissions.CatalogEntry `json:"data"`
	Wildcards []string                   `json:"wildcards"`
}

func (this *ProfileController) GetGrants(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(getGrantsResponse{
		Data:      permissions.Catalog(),
		Wildcards: permissions.Wildcards(),
	})
}

func (this *ProfileController) GetProfiles(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	var query dto.GetProfilesQuery

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	profiles, err := this.profileService.FindAllVisibleTo(currentOrganization.ID, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(profiles)
}

func (this *ProfileController) GetProfile(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	profile, err := this.profileService.FindByIdVisibleTo(ctx.Params("id"), currentOrganization.ID)

	if err != nil {
		return err
	}

	if profile == nil {
		return e.ThrowNotFound("Profile not found in the current organization")
	}

	return ctx.Status(fiber.StatusOK).JSON(profile)
}

func (this *ProfileController) CreateProfile(ctx fiber.Ctx) error {
	var payload dto.CreateProfile

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	profile, err := this.profileService.CreateForOrganization(writeContext(ctx), &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(profile)
}

func (this *ProfileController) UpdateProfile(ctx fiber.Ctx) error {
	var payload dto.UpdateProfile

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	profile, err := this.profileService.UpdateForOrganization(ctx.Params("id"), writeContext(ctx), &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(profile)
}

// Both halves come from OrganizationGuard, which every write below is chained
// behind: the ceiling of whoever is writing is the organization plus its
// participation in it.
func writeContext(ctx fiber.Ctx) *services.ProfileWriteContext {
	return &services.ProfileWriteContext{
		Organization: ctx.Locals("organization").(*entity.Organization),
		Participant:  ctx.Locals("participant").(*entity.Participant),
	}
}

func (this *ProfileController) Register(server *fiber.App) {
	group := server.Group("/core")

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/grants", openapi.Options{
			Summary:     "List the grants this service knows",
			Description: "Lists every grant a profile may be written with, grouped by feature, plus the wildcard forms that are not catalog keys and so cannot be enumerated. A grant is the authoring format of `profiles.permissions`; it is translated into route patterns before any permission is enforced.",
			Tags:        []string{docs.TagGrants},
		}).
			AddResponse(fiber.StatusOK, getGrantsResponse{}, openapi.Options{
				Description: "The grant catalog and the wildcard forms",
			}),
	)

	// Not /core/profiles/grants: fiber matches in registration order, so that path
	// would be swallowed by /core/profiles/:id.
	group.Get(
		"/grants",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetGrants,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/profiles", openapi.Options{
			Summary:     "List profiles",
			Description: "Lists the profiles the current organization may apply: the global ones, seeded for every organization, plus the ones scoped to it. A profile scoped to another organization is never listed, whatever `organization_id` asks for.",
			Tags:        []string{docs.TagProfiles},
		}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Profiles to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Profiles per page - defaults to 10",
			}).
			AddQueryParam("name", openapi.String, openapi.Options{
				Description: "Filters by profile name, case insensitive",
			}).
			AddQueryParam("organization_id", openapi.String, openapi.Options{
				Description: "Narrows the page to the profiles that organization owns, dropping the global ones. It can only narrow what is already visible - any id other than the current organization answers an empty page",
			}).
			AddResponse(fiber.StatusOK, dto.GetProfilesResponse{}, openapi.Options{
				Description: "Page of profiles, each with the permission document as it is stored",
			}),
	)
	group.Get(
		"/profiles",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetProfiles,
	)

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/profiles", openapi.Options{
				Summary:     "Create a profile",
				Description: "Creates a profile scoped to the organization the authenticated user is currently in - only that organization sees it and only it may apply it. The key is generated as `<organization_id>:<NAME_IN_MACRO_CASE>` and never changes afterwards. Only the `grants` half of a permission document is accepted; anything the caller does not itself hold in this organization is refused whole, and nothing is written.",
				Tags:        []string{docs.TagProfiles},
			}),
		).
			AddBody(dto.CreateProfile{}, openapi.Options{
				Required:    true,
				Description: "Name and the grants the profile concedes",
			}).
			AddResponse(fiber.StatusCreated, entity.Profile{}, openapi.Options{
				Description: "The created profile",
			}),
	)
	group.Post(
		"/profiles",
		middleware.BodyValidator[dto.CreateProfile](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.CreateProfile,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/profiles/{id}", openapi.Options{
			Summary:     "Get a profile",
			Description: "Reads a single profile visible to the current organization: a global one, or one scoped to it.",
			Tags:        []string{docs.TagProfiles},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the profile",
			}).
			AddResponse(fiber.StatusOK, entity.Profile{}, openapi.Options{
				Description: "The requested profile",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No profile visible to the current organization matches the given id",
			}),
	)
	group.Get(
		"/profiles/:id",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetProfile,
	)

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "PUT", "/core/profiles/{id}", openapi.Options{
				Summary:     "Update a profile",
				Description: "Edits a profile scoped to the current organization. `key` is never editable, and only the `grants` half of the document is touched - a hand written `api` half is left as it is. Refused with `403` on a global profile, on the `Admin` profile of the organization, on the profile the caller itself participates with, and whenever the requested grants exceed what the caller holds here.",
				Tags:        []string{docs.TagProfiles},
			}),
		).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the profile",
			}).
			AddBody(dto.UpdateProfile{}, openapi.Options{
				Required:    true,
				Description: "Name and grants to write - an absent `permissions` leaves the document untouched",
			}).
			AddResponse(fiber.StatusOK, entity.Profile{}, openapi.Options{
				Description: "The updated profile",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No profile visible to the current organization matches the given id",
			}),
	)
	group.Put(
		"/profiles/:id",
		middleware.BodyValidator[dto.UpdateProfile](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.UpdateProfile,
	)
}
