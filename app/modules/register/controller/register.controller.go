package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/register/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"
	"fmt"

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

func (this *RegisterController) RegisterUser(ctx *fiber.Ctx) error {

	this.logger.Info("Register Controller Triggered")

	app := ctx.Locals("app").(*entity.App)

	appId := app.ID
	usersPoolId := app.UsersPool.ID

	method := ctx.Queries()["method"]

	var payload dto.RegisterPayloadWithPassoword

	ctx.BodyParser(&payload)

	fmt.Println("Payload: ", payload)
	fmt.Println("AppId: ", appId)
	fmt.Println("UsersPoolId: ", usersPoolId)

	if method == "otp" {
		return this.service.RegisterWithOtp()
	}

	if method == "password" || method == "" {

		user, err := this.service.RegisterWithPassword(appId, usersPoolId, payload)

		if err != nil {
			return err
		}

		return ctx.Status(fiber.StatusCreated).JSON(user)
	}

	return e.ThrowUnprocessableEntity("Invalid register method")
}

func (this *RegisterController) Register(server *fiber.App) {
	group := server.Group(
		"/auth",
		this.appGuard.Act,
	)

	group.Post(
		"/register",
		validators.RegisterValidator,
		this.RegisterUser,
	)

}
