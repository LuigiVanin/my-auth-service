package controller

import (
	"auth_service/app/docs"
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	dto "auth_service/app/modules/avatar/models"
	services "auth_service/app/modules/avatar/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/interfaces"
	smodels "auth_service/shared/models"
	"auth_service/shared/storage"

	"github.com/LuigiVanin/openapi-builder/openapi"
	"github.com/gofiber/fiber/v3"
)

type AvatarController struct {
	authGuard              *guards.AuthGuard
	emailVerificationGuard *guards.EmailVerificationGuard
	organizationGuard      *guards.OrganizationGuard
	permissionsGuard       *guards.PermissionsGuard

	avatarService services.IAvatarService
	urls          storage.IUrlResolver

	swagger *openapi.Builder
}

var _ interfaces.IController = &AvatarController{}

func NewAvatarController(
	authGuard *guards.AuthGuard,
	emailVerificationGuard *guards.EmailVerificationGuard,
	organizationGuard *guards.OrganizationGuard,
	permissionsGuard *guards.PermissionsGuard,
	avatarService services.IAvatarService,
	urls storage.IUrlResolver,
	builder *openapi.Builder,
) *AvatarController {
	return &AvatarController{
		authGuard:              authGuard,
		emailVerificationGuard: emailVerificationGuard,
		organizationGuard:      organizationGuard,
		permissionsGuard:       permissionsGuard,
		avatarService:          avatarService,
		urls:                   urls,
		swagger:                builder,
	}
}

func (this *AvatarController) RequestUserUpload(ctx fiber.Ctx) error {
	var payload dto.RequestUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentUser := ctx.Locals("user").(*entity.User)

	grant, err := this.avatarService.RequestUserUpload(currentUser, payload.ContentType)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(grant)
}

func (this *AvatarController) ConfirmUser(ctx fiber.Ctx) error {
	var payload dto.ConfirmUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	user, err := this.avatarService.ConfirmUser(currentUser, currentOrganization, payload.Key)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewUserView(user, this.urls))
}

func (this *AvatarController) RemoveUser(ctx fiber.Ctx) error {
	currentUser := ctx.Locals("user").(*entity.User)
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	user, err := this.avatarService.RemoveUser(currentUser, currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewUserView(user, this.urls))
}

func (this *AvatarController) RequestOrganizationUpload(ctx fiber.Ctx) error {
	var payload dto.RequestUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	grant, err := this.avatarService.RequestOrganizationUpload(ctx.Params("id"), currentOrganization, payload.ContentType)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(grant)
}

func (this *AvatarController) ConfirmOrganization(ctx fiber.Ctx) error {
	var payload dto.ConfirmUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	organization, err := this.avatarService.ConfirmOrganization(ctx.Params("id"), currentOrganization, payload.Key)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewOrganizationView(organization, this.urls))
}

func (this *AvatarController) RemoveOrganization(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	organization, err := this.avatarService.RemoveOrganization(ctx.Params("id"), currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewOrganizationView(organization, this.urls))
}

func (this *AvatarController) RequestAppUpload(ctx fiber.Ctx) error {
	var payload dto.RequestUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	grant, err := this.avatarService.RequestAppUpload(ctx.Params("id"), currentOrganization, payload.ContentType)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(grant)
}

func (this *AvatarController) ConfirmApp(ctx fiber.Ctx) error {
	var payload dto.ConfirmUpload

	if err := ctx.Bind().Body(&payload); err != nil {
		return err
	}

	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	app, err := this.avatarService.ConfirmApp(ctx.Params("id"), currentOrganization, payload.Key)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewAppView(app, this.urls))
}

func (this *AvatarController) RemoveApp(ctx fiber.Ctx) error {
	currentOrganization := ctx.Locals("organization").(*entity.Organization)

	app, err := this.avatarService.RemoveApp(ctx.Params("id"), currentOrganization)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(smodels.NewAppView(app, this.urls))
}

// subjectDocs is what changes between the three subjects; the three routes of
// each are described the same way.
type subjectDocs struct {
	path     string
	tag      string
	noun     string
	response any
	scope    func(route *openapi.RouteBuilder) *openapi.RouteBuilder
}

func (this *AvatarController) Register(server *fiber.App) {
	guarded := func(handler fiber.Handler) []any {
		return []any{this.authGuard.Act, this.emailVerificationGuard.Act, this.organizationGuard.Act, this.permissionsGuard.Act, handler}
	}

	subjects := []struct {
		docs    subjectDocs
		route   string
		request fiber.Handler
		confirm fiber.Handler
		remove  fiber.Handler
	}{
		{
			docs: subjectDocs{
				path:     "/core/users/me",
				tag:      docs.TagUsers,
				noun:     "the authenticated user",
				response: smodels.UserView{},
				scope:    func(route *openapi.RouteBuilder) *openapi.RouteBuilder { return route },
			},
			route:   "/core/users/me",
			request: this.RequestUserUpload,
			confirm: this.ConfirmUser,
			remove:  this.RemoveUser,
		},
		{
			docs: subjectDocs{
				path:     "/core/organizations/{id}",
				tag:      docs.TagOrganizations,
				noun:     "the organization you are currently in",
				response: smodels.OrganizationView{},
				scope: func(route *openapi.RouteBuilder) *openapi.RouteBuilder {
					return route.
						AddPathParam("id", openapi.String, openapi.Options{Required: true, Description: "Id of the organization - it has to be the current one"}).
						AddResponse(fiber.StatusForbidden, e.ProblemDetail{}, openapi.Options{
							Description: "The organization is not the one you are currently in",
						})
				},
			},
			route:   "/core/organizations/:id",
			request: this.RequestOrganizationUpload,
			confirm: this.ConfirmOrganization,
			remove:  this.RemoveOrganization,
		},
		{
			docs: subjectDocs{
				path:     "/core/apps/{id}",
				tag:      docs.TagApps,
				noun:     "an application of the current organization",
				response: smodels.AppView{},
				scope: func(route *openapi.RouteBuilder) *openapi.RouteBuilder {
					return route.
						AddPathParam("id", openapi.String, openapi.Options{Required: true, Description: "Id of the application"}).
						AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{
							Description: "No application of the current organization has this id",
						})
				},
			},
			route:   "/core/apps/:id",
			request: this.RequestAppUpload,
			confirm: this.ConfirmApp,
			remove:  this.RemoveApp,
		},
	}

	for _, subject := range subjects {
		this.document(subject.docs)

		server.Post(subject.route+"/avatar/upload", middleware.BodyValidator[dto.RequestUpload](), guarded(subject.request)...)
		server.Put(subject.route+"/avatar", middleware.BodyValidator[dto.ConfirmUpload](), guarded(subject.confirm)...)

		removeChain := guarded(subject.remove)
		server.Delete(subject.route+"/avatar", removeChain[0], removeChain[1:]...)
	}
}

func (this *AvatarController) document(subject subjectDocs) {
	notConfigured := openapi.Options{Description: "No file storage is configured on this server"}

	this.swagger.Add(
		subject.scope(
			docs.Validated(
				docs.PermissionedRoute(this.swagger, "POST", subject.path+"/avatar/upload", openapi.Options{
					Summary:     "Authorize an avatar upload for " + subject.noun,
					Description: "First of three steps. Answers a presigned POST: send every entry of `fields` as a form field, the file last under `file`, to `url`. The bucket itself refuses a file over `max_bytes`, of another type, or after `expires_at`, and answers `204` on success. Then confirm with `PUT " + subject.path + "/avatar` and the `key` received here.",
					Tags:        []string{subject.tag},
				}),
			),
		).
			AddBody(dto.RequestUpload{}, openapi.Options{Required: true, Description: "Type of the image to be sent"}).
			AddResponse(fiber.StatusCreated, storage.UploadGrant{}, openapi.Options{Description: "The signed upload form"}).
			AddResponse(fiber.StatusNotImplemented, e.ProblemDetail{}, notConfigured),
	)

	this.swagger.Add(
		subject.scope(
			docs.Validated(
				docs.PermissionedRoute(this.swagger, "PUT", subject.path+"/avatar", openapi.Options{
					Summary:     "Confirm an avatar upload for " + subject.noun,
					Description: "Last step. The key has to be one granted for this same subject; the uploaded file is checked against the limits it was signed with, published, and replaces the previous avatar. Each upload yields a new `avatar_url`, so a cached image is never stale.",
					Tags:        []string{subject.tag},
				}),
			),
		).
			AddBody(dto.ConfirmUpload{}, openapi.Options{Required: true, Description: "The `key` answered by the upload authorization"}).
			AddResponse(fiber.StatusOK, subject.response, openapi.Options{Description: "The subject, with its new `avatar_url`"}).
			AddResponse(fiber.StatusBadRequest, e.ProblemDetail{}, openapi.Options{Description: "`key` is not an upload granted for this subject"}).
			AddResponse(fiber.StatusNotFound, e.ProblemDetail{}, openapi.Options{Description: "Nothing was uploaded under `key`: it expired, was never sent, or was already confirmed"}).
			AddResponse(fiber.StatusUnprocessableEntity, e.ProblemDetail{}, openapi.Options{Description: "The uploaded file is over the limit or of a type that is not accepted"}).
			AddResponse(fiber.StatusNotImplemented, e.ProblemDetail{}, notConfigured),
	)

	this.swagger.Add(
		subject.scope(
			docs.PermissionedRoute(this.swagger, "DELETE", subject.path+"/avatar", openapi.Options{
				Summary:     "Remove the avatar of " + subject.noun,
				Description: "Clears the avatar and deletes the image. Idempotent: without an avatar it answers the same body.",
				Tags:        []string{subject.tag},
			}),
		).
			AddResponse(fiber.StatusOK, subject.response, openapi.Options{Description: "The subject, with `avatar_url` null"}).
			AddResponse(fiber.StatusNotImplemented, e.ProblemDetail{}, notConfigured),
	)
}
