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
	smodels "auth_service/shared/models"
	"auth_service/shared/storage"
	"auth_service/shared/utils"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

// resetPasswordResponse mirrors the body returned by ForgotPassword - it only
// exists to describe the payload in the OpenAPI document.
type resetPasswordResponse struct {
	Message string           `json:"message"`
	User    smodels.UserView `json:"user"`
}

// verifyEmailResponse mirrors the body returned by VerifyEmail - it only exists
// to describe the payload in the OpenAPI document.
type verifyEmailResponse struct {
	Message string           `json:"message"`
	User    smodels.UserView `json:"user"`
}

// revokeResponse mirrors the body returned by Revoke - it only exists to
// describe the payload in the OpenAPI document.
type revokeResponse struct {
	Message string `json:"message"`
}

// changeEmailResponse mirrors the body returned by ChangeEmail - it only exists
// to describe the payload in the OpenAPI document.
type changeEmailResponse struct {
	Message string           `json:"message"`
	User    smodels.UserView `json:"user"`
}

var _ interfaces.IController = &AuthorizeController{}

type AuthorizeController struct {
	authService services.IAuthorizeService
	appGuard    *guards.AppGuard
	authGuard   *guards.AuthGuard
	swagger     *openapi.Builder
	urls        storage.IUrlResolver
}

func NewAuthorizeController(
	authorizeService services.IAuthorizeService,
	appGuard *guards.AppGuard,
	authGuard *guards.AuthGuard,
	builder *openapi.Builder,
	urls storage.IUrlResolver,
) *AuthorizeController {
	return &AuthorizeController{
		authService: authorizeService,
		appGuard:    appGuard,
		authGuard:   authGuard,
		swagger:     builder,
		urls:        urls,
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
			"user":    smodels.NewUserView(user, this.urls),
		})
}

func (this *AuthorizeController) Revoke(ctx fiber.Ctx) error {
	sessionId := ctx.Locals("session_id").(string)

	if err := this.authService.Revoke(sessionId); err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(utils.JSON{
		"message": "session revoked",
	})
}

func (this *AuthorizeController) VerifyEmail(ctx fiber.Ctx) error {
	var payload dto.VerifyEmailBody
	if err := ctx.Bind().Body(&payload); err != nil {
		return e.ThrowBadRequest("Invalid request body")
	}

	app := ctx.Locals("app").(*entity.App)
	user := ctx.Locals("user").(*entity.User)

	verified, err := this.authService.VerifyEmail(app, user, payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(utils.JSON{
		"message": "email verified with success",
		"user":    smodels.NewUserView(verified, this.urls),
	})
}

func (this *AuthorizeController) ChangeEmail(ctx fiber.Ctx) error {
	var payload dto.ChangeEmail
	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	app := ctx.Locals("app").(*entity.App)
	user := ctx.Locals("user").(*entity.User)

	updated, err := this.authService.ChangeEmail(app, user, payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(utils.JSON{
		"message": "email changed with success",
		"user":    smodels.NewUserView(updated, this.urls),
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
		docs.AuthRoute(this.swagger, "POST", "/auth/revoke", openapi.Options{
			Summary:     "Revoke the current session",
			Description: "Closes the session the access token belongs to. Both tokens die: the access token because the session stops being active, and the refresh token because its window is closed. Takes no body - the session is the one behind the Authorization header.",
			Tags:        []string{docs.TagAuth},
		}).
			AddResponse(fiber.StatusOK, revokeResponse{}, openapi.Options{
				Description: "Session revoked - revoking an already revoked session answers the same",
			}),
	)
	group.Post("/revoke", this.authGuard.Act, this.Revoke)

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

	// No EmailVerificationGuard, and no /auth route carries one: demanding a
	// verified email in order to change it would lock out exactly the caller who
	// signed up with the wrong address.
	this.swagger.Add(
		docs.Validated(
			docs.AuthRoute(this.swagger, "PUT", "/auth/change_email", openapi.Options{
				Summary:     "Change the email of the authenticated user",
				Description: "Moves the account to a new email address. Requires an OTP generated with the `CHANGE_EMAIL` action through /otp/generate_consumable, sent to and verified against the **new** address. On success the address is marked verified and every session of the caller on this application is closed, so the client has to authenticate again with the new email.",
				Tags:        []string{docs.TagAuth},
			}),
		).
			AddBody(dto.ChangeEmail{}, openapi.Options{
				Required:    true,
				Description: "The new email and the id/code pair of the OTP generated for it",
			}).
			AddResponse(fiber.StatusOK, changeEmailResponse{}, openapi.Options{
				Description: "Email changed - the caller is logged out and the returned user already carries the new address",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "Request body could not be parsed, the OTP code is wrong, already used or expired, or the new email is the one already on the account",
			}).
			AddResponse(fiber.StatusUnauthorized, e.ProblemDetail{}, openapi.Options{
				Description: "Missing or invalid application credentials or access token, or an OTP whose stored email is not the one in the body",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No application matches the given X-Public-Key, or the OTP does not exist",
			}).
			AddResponse(fiber.StatusConflict, e.ProblemDetail{}, openapi.Options{
				Description: "Another user of this pool already uses the requested email",
			}),
	)

	group.Put(
		"/change_email",
		middleware.BodyValidator[dto.ChangeEmail](),
		this.authGuard.Act,
		this.ChangeEmail,
	)
}
