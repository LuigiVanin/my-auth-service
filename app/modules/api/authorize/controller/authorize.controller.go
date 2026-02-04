package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/models/dto"
	"auth_service/app/modules/api/authorize/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"

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

	res, err := this.authService.Authorize(app, authorization, ctx.IP())

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *AuthorizeController) RefreshAuthorization(ctx *fiber.Ctx) error {
	app := ctx.Locals("app").(*entity.App)
	authorization := ctx.Get("Authorization")

	// 1. Try to get token from Header
	if authorization == "" {
		authorization = ctx.Cookies("refresh_token")
	}

	if authorization == "" {
		return e.ThrowBadRequest("Could not find refresh token in the request")
	}

	res, err := this.authService.Refresh(app, authorization, ctx.IP())
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(res)
}

func (this *AuthorizeController) ForgotPassword(ctx *fiber.Ctx) error {
	var payload dto.ResetPasswordPayload
	if err := ctx.BodyParser(&payload); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	app := ctx.Locals("app").(*entity.App)

	user, err := this.authService.ResetPassword(app, payload)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(
		utils.JSON{
			"message": "Password reset successfully",
			"user":    user,
		})
}

func (this *AuthorizeController) Register(server *fiber.App) {
	group := server.Group("/auth")

	group.Post("/authorize", this.AuthorizeRequest)
	group.Post("/refresh", this.RefreshAuthorization)
	group.Put(
		"/forgot_password",
		this.ForgotPassword,
	)
}
