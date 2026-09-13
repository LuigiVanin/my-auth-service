package services

import (
	"encoding/json"
	"fmt"
	"strings"

	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"

	e "auth_service/app/errors"
	ar "auth_service/app/modules/core/app/repository"
	ur "auth_service/app/modules/core/user/repository"
	ups "auth_service/app/modules/core/user_pool/services"
	repo "auth_service/shared/repository"
	"auth_service/shared/tracking"
	"auth_service/shared/utils"

	"go.uber.org/zap"
)

type UserService struct {
	userRepository  ur.IUserRepository
	appRepository   ar.IAppRepository
	userPoolService ups.IUserPoolService
	txManager       repo.ITransactionManager
	logger          *zap.Logger
}

var _ IUserService = &UserService{}

func NewUserService(
	userRepository ur.IUserRepository,
	appRepository ar.IAppRepository,
	userPoolService ups.IUserPoolService,
	txManager repo.ITransactionManager,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userRepository:  userRepository,
		appRepository:   appRepository,
		userPoolService: userPoolService,
		txManager:       txManager,
		logger:          logger,
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

// UpdateForOrganization is the administration write: the target has to sit in a
// pool the current organization owns.
//
// It deliberately does not go through FindById, whose self branch returns before
// the pool check - inheriting it would make editing yourself unrestricted, and
// PUT /core/users/me is the route that is allowed to do that.
func (this *UserService) UpdateForOrganization(
	currentOrganization *entity.Organization,
	targetUserId uint,
	payload *dto.UpdateUser,
) (*entity.User, error) {
	if currentOrganization == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current organization and payload are required")
	}

	stored, err := this.findInOwnedPool(currentOrganization, targetUserId)

	if err != nil {
		return nil, err
	}

	dao := dto.UserUpdateDao{
		Name:             payload.Name,
		Phone:            payload.Phone,
		VerifyEmail:      payload.VerifyEmail,
		TwoFactorEnabled: payload.TwoFactorEnabled,
	}

	if payload.Email != nil {
		email, err := this.availableEmail(*payload.Email, stored)

		if err != nil {
			return nil, err
		}

		dao.Email = email
	}

	if err := this.mergeMetadata(&dao, stored, payload.Metadata); err != nil {
		return nil, err
	}

	return this.applyUpdate(stored, dao)
}

// UpdateSelf needs no scope check: the target is the caller, resolved by the
// AuthGuard and never read from the request.
func (this *UserService) UpdateSelf(
	currentUser *entity.User,
	payload *dto.UpdateUserSelf,
) (*entity.User, error) {
	if currentUser == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current user and payload are required")
	}

	dao := dto.UserUpdateDao{
		Name:  payload.Name,
		Phone: payload.Phone,
	}

	if err := this.mergeMetadata(&dao, currentUser, payload.Metadata); err != nil {
		return nil, err
	}

	return this.applyUpdate(currentUser, dao)
}

func (this *UserService) applyUpdate(stored *entity.User, dao dto.UserUpdateDao) (*entity.User, error) {
	if !repo.HasChanges(dao) {
		return stored, nil
	}

	affected, err := this.userRepository.Update(entity.User{ID: stored.ID}, dao)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to update the user")
	}

	if affected == 0 {
		return nil, e.ThrowNotFound("User not found")
	}

	updated, err := this.userRepository.FindOne(
		entity.User{ID: stored.ID},
		repo.Option{With: []string{"CurrentOrganization", "CurrentOrganization.Profile"}},
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find the user")
	}

	if updated == nil {
		return nil, e.ThrowNotFound("User not found")
	}

	return updated, nil
}

// Track pushes one login onto the queue of the user, dropping whatever no longer
// fits. The row lock is what keeps two concurrent logins of the same user from
// losing one another's event - the document is read, changed and written back, and
// `+ 1` in SQL cannot express a queue.
func (this *UserService) Track(tag tracking.Tag, payload TrackLogin) error {
	if tag != tracking.TagLogin {
		return e.ThrowInternalServerError(fmt.Sprintf("A user does not track `%s`", tag))
	}

	if payload.UserId == 0 {
		return e.ThrowInternalServerError("The user of a login is required")
	}

	tx, err := this.txManager.Tx()

	if err != nil {
		return e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	stored, err := this.userRepository.FindOneForUpdate(payload.UserId, option)

	if err != nil {
		return e.ThrowInternalServerError("Failed to lock the user")
	}

	if stored == nil {
		return e.ThrowNotFound("User not found")
	}

	document, err := tracking.RecordLogin(stored.Tracking, payload.Event)

	if err != nil {
		this.logger.Error(
			"Failed to fold a login into the tracking of a user",
			zap.Uint("user_id", payload.UserId),
			zap.Error(err),
		)

		return e.ThrowInternalServerError("Failed to build the tracking document")
	}

	if _, err := this.userRepository.WriteTracking(payload.UserId, document, option); err != nil {
		return e.ThrowInternalServerError("Failed to write the tracking of the user")
	}

	return tx.Commit()
}

// Tracking is not reachable from here: it is a column of its own, absent from the
// dao, written only by UserRepository.WriteTracking.
func (this *UserService) mergeMetadata(
	dao *dto.UserUpdateDao,
	stored *entity.User,
	patch *json.RawMessage,
) error {
	if patch == nil {
		return nil
	}

	merged, err := utils.MergeJsonPatch(stored.Metadata, *patch)

	if err != nil {
		return e.ThrowBadRequest("`metadata` has to be a JSON object", utils.JSON{"field": "metadata"})
	}

	dao.Metadata = &merged

	return nil
}

// Emails are stored lowercased, and unique per pool. Checked here so a collision
// is a 409 naming the field instead of a unique index violation surfacing as a 500.
func (this *UserService) availableEmail(email string, stored *entity.User) (*string, error) {
	normalized := strings.ToLower(email)

	if normalized == stored.Email {
		return nil, nil
	}

	taken, err := this.FindUserInPool(normalized, stored.UsersPoolId)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to check the email")
	}

	if taken != nil {
		return nil, e.ThrowUserAlreadyExists("Another user of this pool already uses this email")
	}

	return &normalized, nil
}

// The read half of the administration write, with the same folding of "belongs to
// another organization" into "does not exist" that assertPoolInOrganization does.
//
// Preloaded like the read back after the write, so an empty payload answers the
// same shape a real update does.
func (this *UserService) findInOwnedPool(
	currentOrganization *entity.Organization,
	targetUserId uint,
) (*entity.User, error) {
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

	if err := this.assertPoolInOrganization(currentOrganization, user.UsersPoolId); err != nil {
		return nil, err
	}

	return user, nil
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
