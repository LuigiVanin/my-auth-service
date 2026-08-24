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

// NOTE: visibility used to branch on the profile of the caller - ADMIN saw
// everything, MANAGER saw the apps it owned. It is now one question, the same one
// everywhere: does the app belong to the current organization.
func (this *UserService) FindAllUsersFromApp(
	targetAppId string,
	currentUser *entity.User,
	currentApp *entity.App,
	currentOrganization *entity.Organization,
	query *dto.GetUsersAppQuery,
) (*dto.GetUsersAppResponse, error) {

	if currentUser == nil || currentApp == nil || currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current user, app and organization are required")
	}

	app, err := this.appRepository.FindOne(entity.App{ID: targetAppId})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find the app")
	}

	// Folded together: telling "does not exist" apart from "belongs to someone
	// else" turns this into a probe for app ids.
	if app == nil || app.OrganizationId == nil || *app.OrganizationId != currentOrganization.ID {
		return nil, e.ThrowNotFound("App not found in the current organization")
	}

	option := repo.Option{With: []string{"CurrentOrganization", "CurrentOrganization.Profile"}}

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
