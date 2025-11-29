package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/modules/register/services"
	"auth_service/common/interfaces"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type RegisterController struct {
	service  services.IRegisterService
	appGuard *guards.AppGuard
}

var _ interfaces.IController = &RegisterController{}

func NewRegisterController(service services.IRegisterService, appGuard *guards.AppGuard) *RegisterController {
	return &RegisterController{
		service:  service,
		appGuard: appGuard,
	}
}

func (controller *RegisterController) RegisterUser(ctx *fiber.Ctx) error {

	fmt.Println("Register Controller Triggered")
	controller.service.Register()

	return ctx.SendString("Register Controller Triggered")
}

func (controller *RegisterController) Register(server *fiber.App) {
	group := server.Group(
		"/auth",
		controller.appGuard.Act,
	)

	group.Post(
		"/register",
		validators.RegisterValidator,
		controller.RegisterUser,
	)

}
