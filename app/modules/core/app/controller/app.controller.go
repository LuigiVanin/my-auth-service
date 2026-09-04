package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	services "auth_service/app/modules/core/app/services"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type AppController struct {
	appService        services.IAppService
	authGuard         *guards.AuthGuard
	organizationGuard *guards.OrganizationGuard
	permissionsGuard  *guards.PermissionsGuard
	logger            *zap.Logger

	swagger *openapi.Builder
}

var _ interfaces.IController = &AppController{}

func NewAppController(
	authGuard *guards.AuthGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	appService services.IAppService,
	logger *zap.Logger,
	builder *openapi.Builder,
) *AppController {
	return &AppController{
		authGuard:         authGuard,
		organizationGuard: organizationGuard,
		appService:        appService,
		permissionsGuard:  permissionsGuard,
		logger:            logger,
		swagger:           builder,
	}
}

func (this *AppController) CreateApp(ctx fiber.Ctx) error {
	var payload dto.CreateAppPayload

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	app, err := this.appService.CreateWithUserPool(currentUser, currentApp, currentOrganization, &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(app)
}

func (this *AppController) GetApps(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	var query dto.GetAppsQuery
	if err := ctx.Bind().Query(&query); err != nil {
		return e.ThrowBadRequest("Invalid query parameters")
	}

	apps, err := this.appService.FindAll(currentUser, currentApp, currentOrganization, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(apps)
}

func (this *AppController) GetApp(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	app, err := this.appService.FindById(ctx.Params("id"), currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(app)
}

func (this *AppController) UpdateApp(ctx fiber.Ctx) error {
	var payload dto.UpdateApp

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentApp := ctx.Locals("app").(*entity.App)
	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	app, err := this.appService.Update(currentUser, currentApp, currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(app)
}

func (this *AppController) Register(server *fiber.App) {

	group := server.Group("/core")

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/apps", openapi.Options{
				Summary:     "Create an application",
				Description: "Creates an application owned by the authenticated user, as a child of the application the request is made with. The users pool is either an existing one, by id, or a new one, by name.",
				Tags:        []string{docs.TagApps},
			}),
		).
			AddBody(dto.CreateAppPayload{}, openapi.Options{
				Required:    true,
				Description: "Settings of the application - login methods, token type and expiration times",
			}).
			AddResponse(fiber.StatusCreated, entity.App{}, openapi.Options{
				Description: "The created application - `public_key` is the ciphered id to send as X-Public-Key",
			}),
	)
	group.Post(
		"/apps",
		middleware.BodyValidator[dto.CreateAppPayload](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.CreateApp,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/apps", openapi.Options{
			Summary:     "List applications",
			Description: "Lists the applications of the organization the authenticated user is currently in. The scope is the same for every profile - to look at another organization, switch to it with `PUT /core/organizations/switch`.",
			Tags:        []string{docs.TagApps},
		}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Applications to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Applications per page - defaults to 10",
			}).
			AddQueryParam("name", openapi.String, openapi.Options{
				Description: "Filters by application name, case insensitive",
			}).
			AddQueryParam("pool_id", openapi.String, openapi.Options{
				Description: "Lists every application attached to this users pool of the current organization",
			}).
			AddResponse(fiber.StatusOK, dto.GetAppsResponse{}, openapi.Options{
				Description: "Page of applications - the keys of each application are omitted from the listing",
			}),
	)
	group.Get("/apps", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.GetApps)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/apps/{id}", openapi.Options{
			Summary:     "Get an application",
			Description: "Reads a single application of the current organization. `secret_key` is never returned.",
			Tags:        []string{docs.TagApps},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the application",
			}).
			AddResponse(fiber.StatusOK, entity.App{}, openapi.Options{
				Description: "The requested application",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No application of the current organization matches the given id",
			}),
	)
	group.Get("/apps/:id", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.GetApp)

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "PUT", "/core/apps/{id}", openapi.Options{
				Summary:     "Update an application",
				Description: "Updates the settings of an application owned by the authenticated user. Not implemented yet - always answers 500.",
				Tags:        []string{docs.TagApps},
			}),
		).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the application",
			}).
			AddBody(dto.UpdateApp{}, openapi.Options{
				Required:    true,
				Description: "New settings of the application",
			}).
			AddResponse(fiber.StatusOK, entity.App{}, openapi.Options{
				Description: "The updated application",
			}),
	)
	group.Put("/apps/:id", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.UpdateApp)
}
