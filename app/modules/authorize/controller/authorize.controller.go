package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/authorize/models"
	"auth_service/app/modules/authorize/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"
	"auth_service/shared/utils"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

// resetPasswordResponse mirrors the body returned by ForgotPassword - it only
// exists to describe the payload in the OpenAPI document.
type resetPasswordResponse struct {
	Message string      `json:"message"`
	User    entity.User `json:"user"`
}

// verifyEmailResponse mirrors the body returned by VerifyEmail - it only exists
// to describe the payload in the OpenAPI document.
type verifyEmailResponse struct {
	Message string      `json:"message"`
	User    entity.User `json:"user"`
}

var _ interfaces.IController = &AuthorizeController{}

type AuthorizeController struct {
	authService services.IAuthorizeService
	appGuard    *guards.AppGuard
	authGuard   *guards.AuthGuard
	swagger     *openapi.Builder
}

func NewAuthorizeController(
	authorizeService services.IAuthorizeService,
	appGuard *guards.AppGuard,
	authGuard *guards.AuthGuard,
	builder *openapi.Builder,
) *AuthorizeController {
	return &AuthorizeController{
		authService: authorizeService,
		appGuard:    appGuard,
		authGuard:   authGuard,
		swagger:     builder,
	}
}

func (this *AuthorizeController) AuthorizeRequest(ctx fiber.Ctx) error {
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

func (this *AuthorizeController) RefreshAuthorization(ctx fiber.Ctx) error {
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

func (this *AuthorizeController) ForgotPassword(ctx fiber.Ctx) error {
	var payload dto.ResetPasswordPayload
	if err := ctx.Bind().Body(&payload); err != nil {
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

func (this *AuthorizeController) Revoke(ctx fiber.Ctx) error {
	return e.ThrowNotImplementedError("Logout not implemented")
}

func (this *AuthorizeController) VerifyEmail(ctx fiber.Ctx) error {
	var payload dto.VerifyEmailBody
	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	app := ctx.Locals("app").(*entity.App)
	user := ctx.Locals("user").(*entity.User)

	err := this.authService.VerifyEmail(app, user, payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(utils.JSON{
		"message": "email verified with success",
		"user":    user,
	})
}

func (this *AuthorizeController) Register(server *fiber.App) {
	group := server.Group("/auth")

	this.swagger.Add(
		docs.AuthRoute(this.swagger, "POST", "/auth/authorize", openapi.Options{
			Summary:     "Authorize an access token",
			Description: "Validates the access token against the application and the ip of the caller, answering who the token belongs to. This is the endpoint the guards of the service rely on.",
			Tags:        []string{docs.TagAuth},
		}).
			AddResponse(fiber.StatusOK, dto.AuthorizeResponse{}, openapi.Options{
				Description: "Token is valid - carries the user and the session behind it",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "No token was sent in the request",
			}),
	)
	group.Post("/authorize", this.AuthorizeRequest)

	this.swagger.Add(
		docs.AppRoute(this.swagger, "POST", "/auth/refresh", openapi.Options{
			Summary:     "Refresh a session",
			Description: "Exchanges a refresh token for a new pair of tokens. The refresh token is read from the Authorization header and falls back to the `refresh_token` cookie.",
			Tags:        []string{docs.TagAuth},
		}).
			AddHeaderParam("Authorization", openapi.String, openapi.Options{
				Description: "Refresh token returned by /auth/login, /auth/register or a previous refresh - optional when the `refresh_token` cookie is sent",
			}).
			AddResponse(fiber.StatusOK, dto.RefreshResponse{}, openapi.Options{
				Description: "Session refreshed - carries the new access and refresh tokens",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "No refresh token was sent, neither as a header nor as a cookie",
			}),
	)
	group.Post("/refresh", this.RefreshAuthorization)

	this.swagger.Add(
		docs.AppRoute(this.swagger, "PUT", "/auth/forgot_password", openapi.Options{
			Summary:     "Reset a password",
			Description: "Sets a new password for a user of the pool. Requires an OTP generated with the `FORGOT_PASSWORD` action through /otp/generate_consumable.",
			Tags:        []string{docs.TagAuth},
		}).
			AddBody(dto.ResetPasswordPayload{}, openapi.Options{
				Required:    true,
				Description: "Email of the user, the new password and the verified OTP id/code pair",
			}).
			AddResponse(fiber.StatusOK, resetPasswordResponse{}, openapi.Options{
				Description: "Password reset",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "Request body could not be parsed or the OTP is invalid",
			}),
	)
	group.Put(
		"/forgot_password",
		middleware.BodyValidator[dto.ResetPasswordPayload](),
		this.ForgotPassword,
	)

	this.swagger.Add(
		docs.Validated(
			docs.AuthRoute(this.swagger, "PUT", "/auth/verify_email", openapi.Options{
				Summary:     "Verify an email",
				Description: "Marks the email of the authenticated user as verified. Requires an OTP generated with the `VERIFY_EMAIL` action through /otp/generate_consumable, whose stored contact has to be the email sent in the body.",
				Tags:        []string{docs.TagAuth},
			}),
		).
			AddBody(dto.VerifyEmailBody{}, openapi.Options{
				Required:    true,
				Description: "Email being verified and the id/code pair of the OTP generated for it",
			}).
			AddResponse(fiber.StatusCreated, verifyEmailResponse{}, openapi.Options{
				Description: "Email verified - `users.verify_email` is now true",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "Request body could not be parsed, or the OTP code is wrong, already used or expired",
			}).
			AddResponse(fiber.StatusUnauthorized, e.ProblemDetail{}, openapi.Options{
				Description: "Missing or invalid application credentials or access token, or an OTP whose stored email is not the one in the body",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No application matches the given X-Public-Key, the OTP does not exist, or no user of the pool has this email",
			}),
	)

	group.Put(
		"/verify_email",
		middleware.BodyValidator[dto.VerifyEmailBody](),
		this.authGuard.Act,
		this.VerifyEmail,
	)
}
