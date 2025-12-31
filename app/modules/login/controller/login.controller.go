package controller

import (
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/login/services"
	e "auth_service/common/errors"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type LoginController struct {
	service  services.ILoginService
	logger   *zap.Logger
	appGuard *guards.AppGuard
}

var _ interfaces.IController = &LoginController{}

func NewLoginController(service services.ILoginService, appGuard *guards.AppGuard, logger *zap.Logger) *LoginController {
	return &LoginController{
		service:  service,
		appGuard: appGuard,
		logger:   logger,
	}
}

func (this *LoginController) LoginUser(ctx *fiber.Ctx) error {
	this.logger.Info("Register Controller Triggered")

	app := ctx.Locals("app").(*entity.App)

	method := ctx.Queries()["method"]

	var response *dto.LoginResponse
	var err error
	requestInfo := dto.RequestInfo{
		IpAddress: ctx.IP(),
		UserAgent: ctx.Get("User-Agent"),
	}

	if method == "otp" {
		payload, reqErr := parseLoginBody[dto.LoginPayloadWithOtp](ctx)
		if reqErr != nil {
			return reqErr
		}

		response, err = this.service.LoginWithOtp(app, payload, requestInfo)
	}

	if method == "password" || method == "" {
		payload, reqErr := parseLoginBody[dto.LoginPayloadWithPassoword](ctx)
		if reqErr != nil {
			return reqErr
		}

		response, err = this.service.LoginWithPassword(app, payload, requestInfo)
	}

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(response)
}

func (this *LoginController) Register(server *fiber.App) {
	group := server.Group(
		"/auth",
		this.appGuard.Act,
	)

	group.Post(
		"/login",
		validators.LoginValidator,
		this.LoginUser,
	)
}

func parseLoginBody[T any](ctx *fiber.Ctx) (T, error) {
	var payload T
	if err := ctx.BodyParser(&payload); err != nil {
		return payload, e.ThrowBadRequest(err.Error())
	}
	return payload, nil
}
