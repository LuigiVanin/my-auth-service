package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/modules/authorize/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

var _ interfaces.IController = &AuthorizeController{}

type AuthorizeController struct {
	authService services.IAuthorizeService
	appGuard    *guards.AppGuard
}

func NewAuthorizeController(authorizeService services.IAuthorizeService, appGuard *guards.AppGuard) *AuthorizeController {
	return &AuthorizeController{
		authService: authorizeService,
		appGuard:    appGuard,
	}
}

func (this *AuthorizeController) AuthorizeRequest(ctx *fiber.Ctx) error {
	app := ctx.Locals("app").(*entity.App)
	authorization := ctx.Get("Authorization")

	if authorization == "" {
		return e.ThrowBadRequest("Could not find token in the request")
	}

	fmt.Println("Token: ", authorization)
	res, err := this.authService.Authorize(app, authorization, ctx.IP())

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *AuthorizeController) RefreshAuthorization(ctx *fiber.Ctx) error {
	fmt.Println("Refresh Authorization Controller Triggered")

	return e.ThrowNotImplementedError("Refresh Authorization Yet")
}

func (this *AuthorizeController) Register(server *fiber.App) {
	group := server.Group("/auth")

	group.Post("/authorize", this.AuthorizeRequest)
	group.Post("/refresh", this.RefreshAuthorization)
}
