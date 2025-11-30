package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/modules/register/services"
	"auth_service/common/interfaces"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type RegisterController struct {
	service  services.IRegisterService
	logger   *zap.Logger
	appGuard *guards.AppGuard
}

var _ interfaces.IController = &RegisterController{}

func NewRegisterController(service services.IRegisterService, appGuard *guards.AppGuard, logger *zap.Logger) *RegisterController {
	return &RegisterController{
		service:  service,
		appGuard: appGuard,
		logger:   logger,
	}
}

func (controller *RegisterController) RegisterUser(ctx *fiber.Ctx) error {

	controller.logger.Info("Register Controller Triggered")
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
