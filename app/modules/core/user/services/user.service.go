package services

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"strings"

	e "auth_service/app/errors"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user/repository"
	repo "auth_service/shared/repository"
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
	user, err := this.userRepository.FindOne(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err != nil {
		return false, err
	}

	return user != nil, nil
}

// FindUserInPool returns nil, nil when the user is not in the pool.
func (this *UserService) FindUserInPool(email string, usersPoolId string) (*entity.User, error) {
	return this.userRepository.FindOne(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: usersPoolId,
	})
}

func (this *UserService) Update(where entity.User, data dto.UserUpdateDao) (int64, error) {
	return this.userRepository.Update(where, data)
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
		app, err := this.appRepository.FindOne(
			entity.App{ID: targetAppId},
			repo.Option{With: []string{"UsersPool"}},
		)

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

	option := repo.Option{With: []string{"Profile"}}

	if query != nil {
		option = option.Paginate(query.Skip, query.Limit)
	}

	// NOTE: listing and counting are separate calls on purpose, combined here.
	// FindFromAppIdCount ignores the pagination, so the total is the real total.
	users, err := this.userRepository.FindFromAppId(targetAppId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch users")
	}

	total, err := this.userRepository.FindFromAppIdCount(targetAppId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count users")
	}

	return &dto.GetUsersAppResponse{
		Total:  total,
		Amount: len(users),
		Skip:   option.Skip,
		Data:   users,
	}, nil
}
