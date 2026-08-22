package services

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"errors"
	"strings"

	e "auth_service/app/errors"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user/repository"

	"gorm.io/gorm"
)

type UserService struct {
	userRepository ur.IUserRepository
	appRepository  ar.IAppRepository
}

var _ IUserService = &UserService{}

func NewUserService(userRepository ur.IUserRepository, appRepository ar.IAppRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
		appRepository:  appRepository,
	}
}

func (this *UserService) IsAlreadyCreated(email string, app *entity.App) (bool, error) {
	_, err := this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err == nil {
		return true, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return false, err
}

func (this *UserService) FindUserInPool(email string, usersPoolId string) (*entity.User, error) {
	return this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: usersPoolId,
	})
}

func (this *UserService) Update(user *entity.User) error {
	return this.userRepository.Update(user)
}

func (this *UserService) FindAllUsersFromApp(
	targetAppId string,
	currentUser *entity.User,
	currentApp *entity.App,
	query *dto.GetUsersAppQuery,
) (*dto.GetUsersAppResponse, error) {

	if currentUser == nil || currentApp == nil {
		return nil, e.ThrowInternalServerError("Current user or target app is required")
	}

	canAccess := false

	if currentUser.Profile != nil && currentUser.Profile.Key == "ADMIN" {
		canAccess = true

	} else if currentUser.Profile != nil && currentUser.Profile.Key == "MANAGER" {
		app, err := this.appRepository.FindAppbyIdWithPool(targetAppId)

		if err != nil || app == nil {
			return nil, e.ThrowBadRequest("The app doesnt exist")
		}

		var appOwnerUserId uint = 0

		if app.OwnerUserId != nil {
			appOwnerUserId = *app.OwnerUserId
		}

		if appOwnerUserId == currentUser.ID {
			canAccess = true
		}

	}

	if !canAccess {
		return nil, e.ThrowUnauthorizedError("User does not have permission to access users from this app")
	}

	skip := 0
	limit := 10

	if query != nil {
		if query.Skip > 0 {
			skip = query.Skip
		}
		if query.Limit > 0 {
			limit = query.Limit
		}
	}

	users, count, err := this.userRepository.FindFromAppId(targetAppId, skip, limit, "Profile")

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch users")
	}

	return &dto.GetUsersAppResponse{
		Total:  count,
		Amount: len(users),
		Skip:   skip,
		Data:   users,
	}, nil
}
