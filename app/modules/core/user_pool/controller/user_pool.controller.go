package controller

import (
	e "auth_service/app/errors"
	middleware "auth_service/app/middlewares"
	"auth_service/app/middlewares/guards"
	"auth_service/app/models/dto"
	ur "auth_service/app/modules/core/user_pool/repository"
	entity "auth_service/infra/entities"

	"auth_service/common/interfaces"

	"github.com/gofiber/fiber/v2"
)

type UserPoolController struct {
	authGuard          *guards.AuthGuard
	permissionsGuard   *guards.PermissionsGuard
	userPoolRepository ur.IUserPoolRepository
}

var _ interfaces.IController = &UserPoolController{}

func NewUserPoolController(
	authGuard *guards.AuthGuard,
	permissionsGuard *guards.PermissionsGuard,
	userPoolRepository ur.IUserPoolRepository,

) *UserPoolController {
	return &UserPoolController{
		authGuard:          authGuard,
		permissionsGuard:   permissionsGuard,
		userPoolRepository: userPoolRepository,
	}
}

func (this *UserPoolController) CreateUserPool(ctx *fiber.Ctx) error {
	var payload dto.CreateUserPool

	if err := ctx.BodyParser(&payload); err != nil {
		return e.ThrowBadRequest(err.Error())
	}

	// currentApp := ctx.Locals("app").(*entity.App)
	currentUser := ctx.Locals("user").(*entity.User)

	userPool, err := this.userPoolRepository.Create(&entity.UsersPool{
		Name:        payload.Name,
		OwnerUserId: &currentUser.ID,
	})

	if err != nil {

		return e.ThrowInternalServerError("Unable to create users pool")
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":   userPool.ID,
		"name": payload.Name,
	})
}

func (this *UserPoolController) Register(server *fiber.App) {
	group := server.Group("/core/users_pool")

	group.Post("",
		middleware.BodyValidator[dto.CreateUserPool](),
		this.authGuard.Act,
		this.permissionsGuard.Act,
		this.CreateUserPool,
	)

}
