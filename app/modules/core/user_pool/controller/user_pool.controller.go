package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/models/dto"
	ur "auth_service/app/modules/core/user_pool/repository"
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
	authGuard          *guards.AuthGuard
	permissionsGuard   *guards.PermissionsGuard
	userPoolRepository ur.IUserPoolRepository

	swagger *openapi.Builder
}

var _ interfaces.IController = &UserPoolController{}

func NewUserPoolController(
	authGuard *guards.AuthGuard,
	permissionsGuard *guards.PermissionsGuard,
	userPoolRepository ur.IUserPoolRepository,
	builder *openapi.Builder,

) *UserPoolController {
	return &UserPoolController{
		authGuard:          authGuard,
		permissionsGuard:   permissionsGuard,
		userPoolRepository: userPoolRepository,
		swagger:            builder,
	}
}

func (this *UserPoolController) CreateUserPool(ctx fiber.Ctx) error {
	var payload dto.CreateUserPool

	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	// currentApp := ctx.Locals("app").(*entity.App)
	currentUser := ctx.Locals("user").(*entity.User)

	userPool, err := this.userPoolRepository.Create(&entity.UsersPool{
		Name:        payload.Name,
		OwnerUserId: &currentUser.ID,
	})

	if err != nil {

		return e.ThrowInternalServerError("Unable to create users pool")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":   userPool.ID,
		"name": payload.Name,
	})
}

func (this *UserPoolController) Register(server *fiber.App) {
	group := server.Group("/core/users_pool")

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/users_pool", openapi.Options{
				Summary:     "Create a users pool",
				Description: "Creates a users pool owned by the authenticated user. The pool isolates the users of the applications created inside it.",
				Tags:        []string{docs.TagUserPools},
			}),
		).
			AddBody(dto.CreateUserPool{}, openapi.Options{
				Required:    true,
				Description: "Name and description of the pool",
			}).
			AddResponse(fiber.StatusCreated, createUserPoolResponse{}, openapi.Options{
				Description: "The created users pool",
			}),
	)

	group.Post("",
		middleware.BodyValidator[dto.CreateUserPool](),
		this.authGuard.Act,
		this.permissionsGuard.Act,
		this.CreateUserPool,
	)

}
