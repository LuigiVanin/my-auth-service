package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	"auth_service/app/middlewares/guards"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type UserController struct {
	authGuard        *guards.AuthGuard
	permissionsGuard *guards.PermissionsGuard

	logger  *zap.Logger
	swagger *openapi.Builder
}

var _ interfaces.IController = &UserController{}

func NewUserController(
	authGuard *guards.AuthGuard,
	permissionsGuard *guards.PermissionsGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
) *UserController {

	return &UserController{
		authGuard:        authGuard,
		permissionsGuard: permissionsGuard,
		logger:           logger,
		swagger:          builder,
	}
}

func (this *UserController) GetSelf(ctx fiber.Ctx) error {
	return e.ThrowNotImplementedError("GetSelf endpoint not implemented yet")
}

func (this *UserController) Register(server *fiber.App) {
	group := server.Group("/core/users")

	this.swagger.Add(
		docs.AuthRoute(this.swagger, "GET", "/core/users/{id}", openapi.Options{
			Summary:     "Get a user",
			Description: "Reads a single user of the application users pool. Not implemented yet - always answers 501.",
			Tags:        []string{docs.TagUsers},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the user",
			}).
			AddResponse(fiber.StatusOK, entity.User{}, openapi.Options{
				Description: "The requested user",
			}).
			AddResponse(fiber.StatusNotImplemented, e.ProblemDetail{}, openapi.Options{
				Description: "Endpoint not implemented yet",
			}),
	)
	group.Get("/:id", this.authGuard.Act, this.GetSelf)
}
