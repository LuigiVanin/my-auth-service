package controller

import (
	"auth_service/app/apidocs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	services "auth_service/app/modules/core/app/services"
	us "auth_service/app/modules/core/user/services"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// getAppResponse mirrors the body returned by GetApp - it only exists to
// describe the payload in the OpenAPI document.
type getAppResponse struct {
	Message string `json:"message"`
}

type AppController struct {
	appService       services.IAppService
	userService      us.IUserService
	authGuard        *guards.AuthGuard
	permissionsGuard *guards.PermissionsGuard
	logger           *zap.Logger

	swagger *openapi.Builder
}

var _ interfaces.IController = &AppController{}

func NewAppController(
	authGuard *guards.AuthGuard,
	permissionsGuard *guards.PermissionsGuard,
	appService services.IAppService,
	userService us.IUserService,
	logger *zap.Logger,
	builder *openapi.Builder,
) *AppController {
	return &AppController{
		authGuard:        authGuard,
		appService:       appService,
		userService:      userService,
		permissionsGuard: permissionsGuard,
		logger:           logger,
		swagger:          builder,
	}
}

func (this *AppController) CreateApp(ctx fiber.Ctx) error {
	var payload dto.CreateAppPayload

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)

	app, err := this.appService.CreateWithUserPool(currentUser, currentApp, &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(app)
}

func (this *AppController) GetApps(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)

	var query dto.GetAppsQuery
	if err := ctx.Bind().Query(&query); err != nil {
		return e.ThrowBadRequest("Invalid query parameters")
	}

	apps, err := this.appService.FindAll(currentUser, currentApp, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(apps)
}

func (this *AppController) GetUsersFromApp(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)

	targetAppId := ctx.Params("id")

	var query dto.GetUsersAppQuery
	if err := ctx.Bind().Query(&query); err != nil {
		return e.ThrowBadRequest("Invalid query parameters")
	}

	response, err := this.userService.FindAllUsersFromApp(targetAppId, currentUser, currentApp, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(response)
}

func (this *AppController) GetApp(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "App fetched successfully",
	})
}

func (this *AppController) UpdateApp(ctx fiber.Ctx) error {
	var payload dto.UpdateApp

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentApp := ctx.Locals("app").(*entity.App)
	currentUser := ctx.Locals("user").(*entity.User)

	app, err := this.appService.Update(currentUser, currentApp)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(app)
}

func (this *AppController) Register(server *fiber.App) {

	group := server.Group("/core")

	this.swagger.Add(
		apidocs.Validated(
			apidocs.PermissionedRoute(this.swagger, "POST", "/core/apps", openapi.Options{
				Summary:     "Create an application",
				Description: "Creates an application owned by the authenticated user, as a child of the application the request is made with. The users pool is either an existing one, by id, or a new one, by name.",
				Tags:        []string{apidocs.TagApps},
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
		this.permissionsGuard.Act,
		this.CreateApp,
	)

	this.swagger.Add(
		apidocs.PermissionedRoute(this.swagger, "GET", "/core/apps", openapi.Options{
			Summary:     "List applications",
			Description: "Lists the applications owned by the authenticated user. An ADMIN also sees the children of the application the request is made with, and can list the applications of another owner through `owner_user_id`.",
			Tags:        []string{apidocs.TagApps},
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
			AddQueryParam("owner_user_id", openapi.Integer, openapi.Options{
				Description: "Filters by owner - ADMIN profiles only",
			}).
			AddResponse(fiber.StatusOK, dto.GetAppsResponse{}, openapi.Options{
				Description: "Page of applications - the keys of each application are omitted from the listing",
			}),
	)
	group.Get("/apps", this.authGuard.Act, this.permissionsGuard.Act, this.GetApps)

	this.swagger.Add(
		apidocs.PermissionedRoute(this.swagger, "GET", "/core/apps/{id}", openapi.Options{
			Summary:     "Get an application",
			Description: "Reads a single application. Not implemented yet - answers a placeholder message.",
			Tags:        []string{apidocs.TagApps},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the application",
			}).
			AddResponse(fiber.StatusOK, getAppResponse{}, openapi.Options{
				Description: "Placeholder answer",
			}),
	)
	group.Get("/apps/:id", this.authGuard.Act, this.permissionsGuard.Act, this.GetApp)

	this.swagger.Add(
		apidocs.PermissionedRoute(this.swagger, "GET", "/core/apps/{id}/users", openapi.Options{
			Summary:     "List the users of an application",
			Description: "Lists the users of the pool the application belongs to.",
			Tags:        []string{apidocs.TagApps},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the application",
			}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Users to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Users per page - defaults to 10",
			}).
			AddQueryParam("name", openapi.String, openapi.Options{
				Description: "Filters by user name, case insensitive",
			}).
			AddQueryParam("email", openapi.String, openapi.Options{
				Description: "Filters by user email, case insensitive",
			}).
			AddResponse(fiber.StatusOK, dto.GetUsersAppResponse{}, openapi.Options{
				Description: "Page of users",
			}),
	)
	group.Get("/apps/:id/users", this.authGuard.Act, this.permissionsGuard.Act, this.GetUsersFromApp)

	this.swagger.Add(
		apidocs.Validated(
			apidocs.PermissionedRoute(this.swagger, "PUT", "/core/apps/{id}", openapi.Options{
				Summary:     "Update an application",
				Description: "Updates the settings of an application owned by the authenticated user. Not implemented yet - always answers 500.",
				Tags:        []string{apidocs.TagApps},
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
	group.Put("/apps/:id", this.authGuard.Act, this.permissionsGuard.Act, this.UpdateApp)
}
