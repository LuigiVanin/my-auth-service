package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	"auth_service/app/middlewares/guards"
	"auth_service/app/middlewares/validators"
	dto "auth_service/app/modules/register/models"
	"auth_service/app/modules/register/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"
	sharedDto "auth_service/shared/models"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type RegisterController struct {
	service  services.IRegisterService
	logger   *zap.Logger
	appGuard *guards.AppGuard

	swagger *openapi.Builder
}

var _ interfaces.IController = &RegisterController{}

func NewRegisterController(
	service services.IRegisterService,
	appGuard *guards.AppGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
) *RegisterController {
	return &RegisterController{
		service:  service,
		appGuard: appGuard,
		logger:   logger,
		swagger:  builder,
	}
}

func (this *RegisterController) RegisterUser(ctx fiber.Ctx) error {

	this.logger.Info("Register Controller Triggered")

	app := ctx.Locals("app").(*entity.App)

	// Extract request information
	requestInfo := sharedDto.RequestInfo{
		IpAddress: ctx.IP(),
		UserAgent: ctx.Get("User-Agent"),
	}

	method := ctx.Queries()["method"]

	if method == "otp" || method == "" {
		var payload dto.RegisterPayloadWithOtp

		if err := ctx.Bind().Body(&payload); err != nil {
			return e.ThrowBadRequest(err.Error())
		}

		response, err := this.service.RegisterWithOtp(app, payload, requestInfo)

		if err != nil {
			return err
		}

		return ctx.Status(fiber.StatusCreated).JSON(response)
	}

	if method == "password" {
		var payload dto.RegisterPayloadWithPassoword

		if err := ctx.Bind().Body(&payload); err != nil {
			return e.ThrowBadRequest(err.Error())
		}

		response, err := this.service.RegisterWithPassword(app, payload, requestInfo)

		if err != nil {
			return err
		}

		return ctx.Status(fiber.StatusCreated).JSON(response)
	}

	return e.ThrowUnprocessableEntity("Invalid register method")
}

func (this *RegisterController) Register(server *fiber.App) {
	group := server.Group(
		"/auth",
	)

	// The body of the route depends on `method`: the documented schema is the
	// one of the default `otp` method, `password` expects
	// RegisterPayloadWithPassoword (a `password` instead of the `otp` pair).
	this.swagger.Add(
		docs.Validated(
			docs.AppRoute(this.swagger, "POST", "/auth/register", openapi.Options{
				Summary:     "Register a user",
				Description: "Creates a user inside the users pool of the application and already opens a session for it.",
				Tags:        []string{docs.TagAuth},
			}),
		).
			AddQueryParam("method", openapi.String, openapi.Options{
				Description: "`otp` (default) requires an OTP generated through /otp/generate_consumable, `password` registers straight with a password",
			}).
			AddBody(dto.RegisterPayloadWithOtp{}, openapi.Options{
				Required:    true,
				Description: "Data of the new user - shape depends on the `method` query parameter",
			}).
			AddResponse(fiber.StatusCreated, dto.RegisterResponse{}, openapi.Options{
				Description: "User created - carries the access and the refresh token",
			}).
			AddResponse(fiber.StatusConflict, e.ProblemDetail{}, openapi.Options{
				Description: "A user with this email already exists in the pool",
			}),
	)

	group.Post(
		"/register",
		validators.RegisterValidator,
		this.RegisterUser,
	)

}
