package controller

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/models/dto"
	"auth_service/common/interfaces"
	entity "auth_service/infra/entities"

	services "auth_service/app/modules/core/app/services"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AppController struct {
	appService       services.IAppService
	authGuard        *guards.AuthGuard
	permissionsGuard *guards.PermissionsGuard
	logger           *zap.Logger
}

var _ interfaces.IController = &AppController{}

func NewAppController(authGuard *guards.AuthGuard, permissionsGuard *guards.PermissionsGuard, appService services.IAppService, logger *zap.Logger) *AppController {
	return &AppController{
		authGuard:        authGuard,
		appService:       appService,
		permissionsGuard: permissionsGuard,
		logger:           logger,
	}
}

func (this *AppController) CreateApp(ctx *fiber.Ctx) error {
	var payload dto.CreateAppPayload

	if err := ctx.BodyParser(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	currentUser := ctx.Locals("user").(*entity.User)
	currentApp := ctx.Locals("app").(*entity.App)

	app, err := this.appService.CreateWithUserPool(currentUser, currentApp, &payload)

	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(app)
}

func (this *AppController) GetApps(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Apps fetched successfully",
	})
}

func (this *AppController) GetApp(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "App fetched successfully",
	})
}

func (this *AppController) GetUsersApps(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Users apps fetched successfully",
	})
}

func (this *AppController) UpdateApp(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "App updated successfully",
	})
}

func (this *AppController) Register(server *fiber.App) {

	group := server.Group("/core")

	group.Post(
		"/apps",
		middleware.BodyValidator[dto.CreateAppPayload](),
		this.authGuard.Act,
		this.permissionsGuard.Act,
		this.CreateApp,
	)
	group.Get("/apps", this.authGuard.Act, this.permissionsGuard.Act, this.GetApps)
	group.Get("/apps/:id", this.authGuard.Act, this.permissionsGuard.Act, this.GetApp)
	group.Get("/user/:user_id/apps", this.authGuard.Act, this.permissionsGuard.Act, this.GetUsersApps)
	group.Put("/apps/:id", this.authGuard.Act, this.permissionsGuard.Act, this.UpdateApp)
}
