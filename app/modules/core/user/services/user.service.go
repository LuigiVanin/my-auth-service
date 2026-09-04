package services

import (
	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	"strings"

	e "auth_service/app/errors"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user/repository"
	ups "auth_service/app/modules/core/user_pool/services"
	repo "auth_service/shared/repository"
)

type UserService struct {
	userRepository  ur.IUserRepository
	appRepository   ar.IAppRepository
	userPoolService ups.IUserPoolService
}

var _ IUserService = &UserService{}

func NewUserService(
	userRepository ur.IUserRepository,
	appRepository ar.IAppRepository,
	userPoolService ups.IUserPoolService,
) *UserService {
	return &UserService{
		userRepository:  userRepository,
		appRepository:   appRepository,
		userPoolService: userPoolService,
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

// assertPoolInOrganization is the visibility rule every user read goes through:
// the users of a pool are readable only by the organization that owns the pool.
//
// "Belongs to another organization" is folded into "does not exist", so a caller
// cannot use the error to probe for pool ids.
func (this *UserService) assertPoolInOrganization(
	currentOrganization *entity.Organization,
	usersPoolId string,
) error {
	pool, err := this.userPoolService.FindByIdInOrganization(usersPoolId, currentOrganization.ID)

	if err != nil {
		return e.ThrowInternalServerError("Failed to find the users pool")
	}

	if pool == nil {
		return e.ThrowNotFound("Users pool not found in the current organization")
	}

	return nil
}

// resolveListPool turns the filter of the listing into the id of a pool the
// current organization owns.
func (this *UserService) resolveListPool(
	currentOrganization *entity.Organization,
	query *dto.UserListQuery,
) (string, error) {
	hasApp := query.AppId != ""
	hasPool := query.PoolId != ""

	switch {
	case hasApp && hasPool:
		return "", e.ThrowBadRequest("Only one of `app_id` and `pool_id` can be used at a time")

	case hasPool:
		if err := this.assertPoolInOrganization(currentOrganization, query.PoolId); err != nil {
			return "", err
		}

		return query.PoolId, nil

	case hasApp:
		app, err := this.appRepository.FindOne(entity.App{ID: query.AppId})

		if err != nil {
			return "", e.ThrowInternalServerError("Failed to find the app")
		}

		if app == nil || app.OrganizationId == nil || *app.OrganizationId != currentOrganization.ID {
			return "", e.ThrowNotFound("App not found in the current organization")
		}

		return app.UsersPoolId, nil

	default:
		// Rejected rather than defaulted: a listing must never fall back to
		// every user in the database.
		return "", e.ThrowBadRequest("One of `app_id` or `pool_id` is required")
	}
}

func (this *UserService) List(
	currentUser *entity.User,
	currentOrganization *entity.Organization,
	query *dto.UserListQuery,
) (*dto.GetUsersResponse, error) {
	if currentUser == nil || currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current user and organization are required")
	}

	if query == nil {
		return nil, e.ThrowBadRequest("One of `app_id` or `pool_id` is required")
	}

	usersPoolId, err := this.resolveListPool(currentOrganization, query)

	if err != nil {
		return nil, err
	}

	search := ur.UserSearch{
		UsersPoolId: usersPoolId,
		Name:        query.Name,
		Email:       query.Email,
	}

	option := repo.Option{
		With: []string{"CurrentOrganization", "CurrentOrganization.Profile"},
	}.Paginate(query.Skip, query.Limit)

	users, err := this.userRepository.FindSearch(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch users")
	}

	total, err := this.userRepository.FindSearchCount(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count users")
	}

	return &dto.GetUsersResponse{
		Total:  total,
		Amount: len(users),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   users,
	}, nil
}

func (this *UserService) FindById(
	currentUser *entity.User,
	currentOrganization *entity.Organization,
	targetUserId uint,
) (*entity.User, error) {
	if currentUser == nil || currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current user and organization are required")
	}

	user, err := this.userRepository.FindOne(
		entity.User{ID: targetUserId},
		repo.Option{With: []string{"CurrentOrganization", "CurrentOrganization.Profile"}},
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find the user")
	}

	if user == nil {
		return nil, e.ThrowNotFound("User not found")
	}

	// /core/users/me resolves here: a user always reads itself.
	if user.ID == currentUser.ID {
		return user, nil
	}

	if err := this.assertPoolInOrganization(currentOrganization, user.UsersPoolId); err != nil {
		return nil, err
	}

	return user, nil
}
