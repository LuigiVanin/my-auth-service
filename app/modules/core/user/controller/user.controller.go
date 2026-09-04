package controller

import (
	"strconv"

	"auth_service/app/docs"
	e "auth_service/app/errors"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	us "auth_service/app/modules/core/user/services"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type UserController struct {
	authGuard         *guards.AuthGuard
	organizationGuard *guards.OrganizationGuard
	permissionsGuard  *guards.PermissionsGuard

	userService us.IUserService

	logger  *zap.Logger
	swagger *openapi.Builder
}

var _ interfaces.IController = &UserController{}

func NewUserController(
	authGuard *guards.AuthGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
	userService us.IUserService,
) *UserController {

	return &UserController{
		authGuard:         authGuard,
		organizationGuard: organizationGuard,
		permissionsGuard:  permissionsGuard,
		userService:       userService,
		logger:            logger,
		swagger:           builder,
	}
}

// GetSelf is /core/users/{id} on the id of the authenticated user.
func (this *UserController) GetSelf(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	user, err := this.userService.FindById(currentUser, currentOrganization, currentUser.ID)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(user)
}

func (this *UserController) GetById(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	targetUserId, parseError := strconv.ParseUint(ctx.Params("id"), 10, 64)

	if parseError != nil {
		return e.ThrowBadRequest("The id of the user has to be a positive number")
	}

	user, err := this.userService.FindById(currentUser, currentOrganization, uint(targetUserId))

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(user)
}

func (this *UserController) List(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	query := dto.UserListQuery{}

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	list, err := this.userService.List(currentUser, currentOrganization, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(list)
}

func (this *UserController) Register(server *fiber.App) {
	group := server.Group("/core/users")

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/users", openapi.Options{
			Summary:     "List users",
			Description: "Lists the users of one users pool, named either by `pool_id` or by the `app_id` of an application attached to it - exactly one of the two is required. The caller has to own the pool.",
			Tags:        []string{docs.TagUsers},
		}).
			AddQueryParam("app_id", openapi.String, openapi.Options{
				Description: "Lists the users of the pool this application belongs to - mutually exclusive with `pool_id`",
			}).
			AddQueryParam("pool_id", openapi.String, openapi.Options{
				Description: "Lists the users of this pool - mutually exclusive with `app_id`",
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
			AddResponse(fiber.StatusOK, dto.GetUsersResponse{}, openapi.Options{
				Description: "Page of users",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No application or users pool matches the given filter",
			}),
	)
	group.Get("", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.List)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/users/me", openapi.Options{
			Summary:     "Get the authenticated user",
			Description: "Shortcut for `/core/users/{id}` on the id of the authenticated user.",
			Tags:        []string{docs.TagUsers},
		}).
			AddResponse(fiber.StatusOK, entity.User{}, openapi.Options{
				Description: "The authenticated user",
			}),
	)
	// Before /:id - fiber matches in registration order, so the literal has to
	// come first or `me` is read as an id.
	group.Get("/me", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.GetSelf)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/users/{id}", openapi.Options{
			Summary:     "Get a user",
			Description: "Reads a single user. The caller has to own the users pool the target user belongs to, unless the target is the caller itself.",
			Tags:        []string{docs.TagUsers},
		}).
			AddPathParam("id", openapi.Integer, openapi.Options{
				Required:    true,
				Description: "Id of the user",
			}).
			AddResponse(fiber.StatusOK, entity.User{}, openapi.Options{
				Description: "The requested user",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No user matches the given id",
			}),
	)
	group.Get("/:id", this.authGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, this.GetById)
}
