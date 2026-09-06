package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/user_pool/models"
	ups "auth_service/app/modules/core/user_pool/services"
	entity "auth_service/infra/entities"

	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

// createUserPoolResponse mirrors the body returned by CreateUserPool - it only
// exists to describe the payload in the OpenAPI document.
type createUserPoolResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type UserPoolController struct {
	authGuard         *guards.AuthGuard
	organizationGuard *guards.OrganizationGuard
	permissionsGuard  *guards.PermissionsGuard
	userPoolService   ups.IUserPoolService

	swagger *openapi.Builder
}

var _ interfaces.IController = &UserPoolController{}

func NewUserPoolController(
	authGuard *guards.AuthGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	userPoolService ups.IUserPoolService,
	builder *openapi.Builder,

) *UserPoolController {
	return &UserPoolController{
		authGuard:         authGuard,
		organizationGuard: organizationGuard,
		permissionsGuard:  permissionsGuard,
		userPoolService:   userPoolService,
		swagger:           builder,
	}
}

func (this *UserPoolController) CreateUserPool(ctx fiber.Ctx) error {
	var payload dto.CreateUserPool

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	userPool, err := this.userPoolService.Create(ups.CreateUserPoolData{
		Name:             payload.Name,
		OwnerUserId:      &currentUser.ID,
		OrganizationId:   &currentOrganization.ID,
		DefaultProfileId: payload.DefaultProfileId,
	}, currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":   userPool.ID,
		"name": userPool.Name,
	})
}

func (this *UserPoolController) List(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	query := dto.UserPoolListQuery{}

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	pools, err := this.userPoolService.List(currentOrganization, &query)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(pools)
}

func (this *UserPoolController) GetUserPool(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	pool, err := this.userPoolService.FindByIdInOrganization(ctx.Params("id"), currentOrganization.ID)

	if err != nil {
		return e.ThrowInternalServerError("Failed to find the users pool")
	}

	if pool == nil {
		return e.ThrowNotFound("Users pool not found in the current organization")
	}

	return ctx.Status(fiber.StatusOK).JSON(pool)
}

func (this *UserPoolController) Register(server *fiber.App) {
	group := server.Group("/core/users_pool")

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/users_pool", openapi.Options{
				Summary:     "Create a users pool",
				Description: "Creates a users pool owned by the organization the authenticated user is currently in. The pool isolates the users of the applications created inside it, and its `default_profile_id` is the permission ceiling every organization born in it receives - it can never grant more than the organization creating the pool holds.",
				Tags:        []string{docs.TagUserPools},
			}),
		).
			AddBody(dto.CreateUserPool{}, openapi.Options{
				Required:    true,
				Description: "Name, description and the permission ceiling of the organizations born in the pool",
			}).
			AddResponse(fiber.StatusCreated, createUserPoolResponse{}, openapi.Options{
				Description: "The created users pool",
			}),
	)

	group.Post("",
		middleware.BodyValidator[dto.CreateUserPool](),
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.CreateUserPool,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/users_pool", openapi.Options{
			Summary:     "List users pools",
			Description: "Lists the users pools of the organization the authenticated user is currently in. To look at another organization, switch to it with `PUT /core/organizations/switch`.",
			Tags:        []string{docs.TagUserPools},
		}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Pools to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Pools per page - defaults to 10",
			}).
			AddQueryParam("name", openapi.String, openapi.Options{
				Description: "Filters by pool name, case insensitive",
			}).
			AddResponse(fiber.StatusOK, dto.GetUserPoolsResponse{}, openapi.Options{
				Description: "Page of users pools",
			}),
	)
	group.Get("",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.List,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/users_pool/{id}", openapi.Options{
			Summary:     "Get a users pool",
			Description: "Reads a single users pool of the current organization.",
			Tags:        []string{docs.TagUserPools},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the users pool",
			}).
			AddResponse(fiber.StatusOK, entity.UsersPool{}, openapi.Options{
				Description: "The requested users pool",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No users pool of the current organization matches the given id",
			}),
	)
	group.Get("/:id",
		this.authGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetUserPool,
	)
}
