package controller

import (
	e "auth_service/app/errors"
	"auth_service/app/middlewares/guards"
	"auth_service/common/interfaces"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type UserController struct {
	authGuard        *guards.AuthGuard
	permissionsGuard *guards.PermissionsGuard
	logger           *zap.Logger
}

var _ interfaces.IController = &UserController{}

func NewUserController(
	authGuard *guards.AuthGuard,
	permissionsGuard *guards.PermissionsGuard,
	logger *zap.Logger,
) *UserController {
	return &UserController{
		authGuard:        authGuard,
		permissionsGuard: permissionsGuard,
		logger:           logger,
	}
}

func (this *UserController) GetSelf(ctx *fiber.Ctx) error {
	return e.ThrowNotImplementedError("GetSelf endpoint not implemented yet")
}

func (this *UserController) Register(server *fiber.App) {
	group := server.Group("/core/users")

	group.Get("/self", this.authGuard.Act, this.GetSelf)
}
