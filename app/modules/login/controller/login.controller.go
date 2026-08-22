package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	"auth_service/app/models/dto"
	"auth_service/app/modules/login/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type LoginController struct {
	service  services.ILoginService
	logger   *zap.Logger
	appGuard *guards.AppGuard

	swagger *openapi.Builder
}

var _ interfaces.IController = &LoginController{}

func NewLoginController(
	service services.ILoginService,
	appGuard *guards.AppGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
) *LoginController {
	return &LoginController{
		service:  service,
		appGuard: appGuard,
		logger:   logger,
		swagger:  builder,
	}
}

func (this *LoginController) LoginUser(ctx fiber.Ctx) error {
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
	)

	// The body of the route depends on `method`: the documented schema is the
	// one of the default `password` method, `otp` expects LoginPayloadWithOtp
	// (`email` plus the `otp` id/code pair) instead.
	this.swagger.Add(
		docs.Validated(
			docs.AppRoute(this.swagger, "POST", "/auth/login", openapi.Options{
				Summary:     "Log a user in",
				Description: "Opens a session for a user of the application users pool and returns the access and refresh tokens.",
				Tags:        []string{docs.TagAuth},
			}),
		).
			AddQueryParam("method", openapi.String, openapi.Options{
				Description: "`password` (default) authenticates with email and password, `otp` with an OTP previously verified through /otp/generate_consumable",
			}).
			AddBody(dto.LoginPayloadWithPassoword{}, openapi.Options{
				Required:    true,
				Description: "Credentials of the user - shape depends on the `method` query parameter",
			}).
			AddResponse(fiber.StatusOK, dto.LoginResponse{}, openapi.Options{
				Description: "Session opened - carries the access and the refresh token",
			}),
	)

	group.Post(
		"/login",
		validators.LoginValidator,
		this.LoginUser,
	)
}

func parseLoginBody[T any](ctx fiber.Ctx) (T, error) {
	var payload T
	if err := ctx.Bind().Body(&payload); err != nil {
		return payload, e.ThrowBadRequest(err.Error())
	}
	return payload, nil
}
