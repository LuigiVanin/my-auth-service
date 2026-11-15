package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/core/invite/models"
	"auth_service/app/modules/core/invite/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type InviteController struct {
	inviteService services.IInviteService

	authGuard              *guards.AuthGuard
	emailVerificationGuard *guards.EmailVerificationGuard
	organizationGuard      *guards.OrganizationGuard
	permissionsGuard       *guards.PermissionsGuard

	logger  *zap.Logger
	swagger *openapi.Builder
}

var _ interfaces.IController = &InviteController{}

func NewInviteController(
	inviteService services.IInviteService,
	authGuard *guards.AuthGuard,
	emailVerificationGuard *guards.EmailVerificationGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	logger *zap.Logger,
	builder *openapi.Builder,
) *InviteController {
	return &InviteController{
		inviteService:          inviteService,
		authGuard:              authGuard,
		emailVerificationGuard: emailVerificationGuard,
		organizationGuard:      organizationGuard,
		permissionsGuard:       permissionsGuard,
		logger:                 logger,
		swagger:                builder,
	}
}

func (this *InviteController) CreateInvite(ctx fiber.Ctx) error {
	var payload dto.CreateInvite

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	invite, err := this.inviteService.Create(
		ctx.Params("id"),
		ctx.Locals("organization").(*entity.Organization),
		ctx.Locals("participant").(*entity.Participant),
		ctx.Locals("user").(*entity.User),
		ctx.Locals("app").(*entity.App),
		&payload,
	)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(invite)
}

func (this *InviteController) GetInvites(ctx fiber.Ctx) error {
	var query dto.GetInvitesQuery

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	invites, err := this.inviteService.List(
		ctx.Params("id"),
		ctx.Locals("organization").(*entity.Organization),
		&query,
	)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(invites)
}

func (this *InviteController) RevokeInvite(ctx fiber.Ctx) error {
	invite, err := this.inviteService.Revoke(
		ctx.Params("id"),
		ctx.Params("invite_id"),
		ctx.Locals("organization").(*entity.Organization),
	)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(invite)
}

func (this *InviteController) DescribeInvite(ctx fiber.Ctx) error {
	var query dto.DescribeInviteQuery

	if err := ctx.Bind().Query(&query); err != nil {
		return err
	}

	description, err := this.inviteService.Describe(query.Token, ctx.Locals("app").(*entity.App))

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(description)
}

func (this *InviteController) AcceptInvite(ctx fiber.Ctx) error {
	var payload dto.AcceptInvite

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	accepted, err := this.inviteService.Accept(payload.Token, ctx.Locals("app").(*entity.App))

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(accepted)
}

func (this *InviteController) Register(server *fiber.App) {
	core := server.Group("/core")

	this.swagger.Add(
		docs.Validated(
			docs.PermissionedRoute(this.swagger, "POST", "/core/organizations/{id}/invites", openapi.Options{
				Summary:     "Invite someone to an organization",
				Description: "Emails a link that adds the address to the organization with the chosen profile. `id` has to be the organization the authenticated user is currently in. The profile has to be visible to the organization and grant no more than the caller holds here. `ttl_seconds` (1 hour to 30 days) falls back to the organization's `invite_ttl_seconds`, then to the service default of 7 days. A pending invite for the same address is revoked and replaced. The token is never returned - it only travels in the email.",
				Tags:        []string{docs.TagInvites},
			}),
		).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the organization, which has to be the current one",
			}).
			AddBody(dto.CreateInvite{}, openapi.Options{
				Required:    true,
				Description: "Address, profile and optional validity of the invite",
			}).
			AddResponse(fiber.StatusCreated, dto.InviteResponse{}, openapi.Options{
				Description: "The invite, pending",
			}).
			AddResponse(fiber.StatusConflict, e.ProblemDetail{}, openapi.Options{
				Description: "The address already participates in this organization",
			}).
			AddResponse(fiber.StatusNotImplemented, e.ProblemDetail{}, openapi.Options{
				Description: "No invite accept page is configured on this server",
			}),
	)
	core.Post(
		"/organizations/:id/invites",
		middleware.BodyValidator[dto.CreateInvite](),
		this.authGuard.Act,
		this.emailVerificationGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.CreateInvite,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "GET", "/core/organizations/{id}/invites", openapi.Options{
			Summary:     "List the invites of an organization",
			Description: "Lists the invites of the organization the authenticated user is currently in, newest first, each with its status. `id` has to be that organization.",
			Tags:        []string{docs.TagInvites},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the organization, which has to be the current one",
			}).
			AddQueryParam("status", openapi.String, openapi.Options{
				Description: "`pending`, `accepted`, `revoked` or `expired` - every invite when absent",
			}).
			AddQueryParam("skip", openapi.Integer, openapi.Options{
				Description: "Invites to skip - defaults to 0",
			}).
			AddQueryParam("limit", openapi.Integer, openapi.Options{
				Description: "Invites per page - defaults to 10",
			}).
			AddResponse(fiber.StatusOK, dto.GetInvitesResponse{}, openapi.Options{
				Description: "Page of invites, each with its profile and status",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "Unknown `status`",
			}),
	)
	core.Get(
		"/organizations/:id/invites",
		this.authGuard.Act,
		this.emailVerificationGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.GetInvites,
	)

	this.swagger.Add(
		docs.PermissionedRoute(this.swagger, "DELETE", "/core/organizations/{id}/invites/{invite_id}", openapi.Options{
			Summary:     "Revoke an invite",
			Description: "Revokes a pending invite: its link stops working at once. `id` has to be the organization the authenticated user is currently in.",
			Tags:        []string{docs.TagInvites},
		}).
			AddPathParam("id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the organization, which has to be the current one",
			}).
			AddPathParam("invite_id", openapi.String, openapi.Options{
				Required:    true,
				Description: "Id of the invite",
			}).
			AddResponse(fiber.StatusOK, dto.InviteResponse{}, openapi.Options{
				Description: "The revoked invite",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "No invite of this organization matches the given id",
			}).
			AddResponse(fiber.StatusConflict, e.ProblemDetail{}, openapi.Options{
				Description: "The invite is not pending - already accepted, revoked or expired",
			}),
	)
	core.Delete(
		"/organizations/:id/invites/:invite_id",
		this.authGuard.Act,
		this.emailVerificationGuard.Act,
		this.organizationGuard.Act,
		this.permissionsGuard.Act,
		this.RevokeInvite,
	)

	auth := server.Group("/auth")

	this.swagger.Add(
		docs.AppRoute(this.swagger, "GET", "/auth/invite", openapi.Options{
			Summary:     "Describe an invite",
			Description: "Tells the bearer of an invite token which organization, profile and address it is for, until when it is valid, and whether the address already has an account in the pool - `true` means accept with `POST /auth/invite/accept`, `false` means sign up with `POST /auth/register?method=invite`. Only the application that issued the invite can read it.",
			Tags:        []string{docs.TagInvites},
		}).
			AddQueryParam("token", openapi.String, openapi.Options{
				Required:    true,
				Description: "The token of the invite link",
			}).
			AddResponse(fiber.StatusOK, dto.InviteDescription{}, openapi.Options{
				Description: "What the invite is for",
			}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{
				Description: "The invite was already accepted, was revoked or replaced, or has expired - `detail` says which",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "The token matches no invite of this application",
			}),
	)
	auth.Get("/invite", this.DescribeInvite)

	this.swagger.Add(
		docs.Validated(
			docs.AppRoute(this.swagger, "POST", "/auth/invite/accept", openapi.Options{
				Summary:     "Accept an invite",
				Description: "Adds the account the invite was sent to into the organization, with the profile of the invite. The token is the only proof needed - no session. The address has to have an account in the pool; otherwise sign up with `POST /auth/register?method=invite`. Accepting marks the email verified and does not change the current organization of the account - use `PUT /core/organizations/switch` for that. Someone who already participates keeps their profile and the invite is consumed.",
				Tags:        []string{docs.TagInvites},
			}),
		).
			AddBody(dto.AcceptInvite{}, openapi.Options{
				Required:    true,
				Description: "The token of the invite link",
			}).
			AddResponse(fiber.StatusOK, dto.AcceptInviteResponse{}, openapi.Options{
				Description: "The organization joined and the participation in it",
			}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
				Description: "The token matches no invite of this application, or the address has no account in the pool",
			}),
	)
	auth.Post(
		"/invite/accept",
		middleware.BodyValidator[dto.AcceptInvite](),
		this.AcceptInvite,
	)
}
