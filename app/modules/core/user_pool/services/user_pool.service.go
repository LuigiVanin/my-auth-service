package services

import (
	"encoding/json"
	"fmt"

	e "auth_service/app/errors"
	prs "auth_service/app/modules/core/profile/services"
	dto "auth_service/app/modules/core/user_pool/models"
	upr "auth_service/app/modules/core/user_pool/repository"
	"auth_service/app/modules/utils/cipher"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"
	"auth_service/shared/tracking"
	"auth_service/shared/utils"

	"go.uber.org/zap"
)

type UserPoolService struct {
	userPoolRepository upr.IUserPoolRepository
	profileService     prs.IProfileService
	cipherService      cipher.ICipherService
	txManager          repo.ITransactionManager
	logger             *zap.Logger
}

var _ IUserPoolService = &UserPoolService{}

func NewUserPoolService(
	userPoolRepository upr.IUserPoolRepository,
	profileService prs.IProfileService,
	cipherService cipher.ICipherService,
	txManager repo.ITransactionManager,
	logger *zap.Logger,
) *UserPoolService {
	return &UserPoolService{
		userPoolRepository: userPoolRepository,
		profileService:     profileService,
		cipherService:      cipherService,
		txManager:          txManager,
		logger:             logger,
	}
}

// Create inserts the pool and stamps its public key, which is derived from the
// id and therefore only knowable after the insert.
//
// NOTE: this used to live inside UserPoolRepository, which meant the repository
// depended on the cipher service and rewrote the whole row with Save. Deriving
// the key is business logic, so it belongs here; the repository now only writes
// the one column that changed.
func (this *UserPoolService) Create(
	data CreateUserPoolData,
	granter *entity.Organization,
	caller *entity.Participant,
) (*entity.UsersPool, error) {
	defaultProfile, err := this.resolveDefaultProfile(data.DefaultProfileId, granter, caller)

	if err != nil {
		return nil, err
	}

	pool, err := this.userPoolRepository.Create(entity.UsersPool{
		Name:             data.Name,
		Description:      data.Description,
		OwnerUserId:      data.OwnerUserId,
		OrganizationId:   data.OrganizationId,
		DefaultProfileId: defaultProfile.ID,
		Metadata:         json.RawMessage(`{}`),
	})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create users pool")
	}

	publicKey, err := this.cipherService.EncryptUuidIntoToken(pool.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt the users pool public key")
	}

	if _, err := this.userPoolRepository.Update(
		entity.UsersPool{ID: pool.ID},
		dto.UserPoolUpdateDao{PublicKey: &publicKey},
	); err != nil {
		return nil, e.ThrowInternalServerError("Unable to update users pool after creation")
	}

	pool.PublicKey = publicKey
	pool.DefaultProfile = defaultProfile

	return pool, nil
}

func (this *UserPoolService) FindById(id string) (*entity.UsersPool, error) {
	return this.userPoolRepository.FindOne(
		entity.UsersPool{ID: id},
		repo.Option{With: []string{"DefaultProfile"}},
	)
}

// Folds "belongs to another organization" into "does not exist", so a caller
// cannot probe for pool ids.
func (this *UserPoolService) FindByIdInOrganization(
	id string,
	organizationId string,
) (*entity.UsersPool, error) {
	pool, err := this.FindById(id)

	if err != nil {
		return nil, err
	}

	if pool == nil || pool.OrganizationId == nil || *pool.OrganizationId != organizationId {
		return nil, nil
	}

	return pool, nil
}

func (this *UserPoolService) List(
	currentOrganization *entity.Organization,
	query *dto.UserPoolListQuery,
) (*dto.GetUserPoolsResponse, error) {
	if currentOrganization == nil {
		return nil, e.ThrowInternalServerError("Current organization is required")
	}

	search := upr.UserPoolSearch{OrganizationId: currentOrganization.ID}

	option := repo.Option{With: []string{"DefaultProfile"}}

	if query != nil {
		search.Name = query.Name
		option = option.Paginate(query.Skip, query.Limit)
	}

	pools, err := this.userPoolRepository.FindSearch(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch users pools")
	}

	total, err := this.userPoolRepository.FindSearchCount(search, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count users pools")
	}

	for idx := range pools {
		// Ten months of counters per row would be a page nobody reads. The GET of one
		// pool is where the tracking is reported.
		pools[idx].Tracking = nil
	}

	return &dto.GetUserPoolsResponse{
		Total:  total,
		Amount: len(pools),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   pools,
	}, nil
}

// "Belongs to another organization" is a 404 here too: the scoped lookup already
// folds it into "does not exist", and answering 403 would turn the id into an
// oracle for the pools of other organizations.
func (this *UserPoolService) Update(
	id string,
	granter *entity.Organization,
	caller *entity.Participant,
	payload *dto.UpdateUserPool,
) (*entity.UsersPool, error) {
	if granter == nil || caller == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Granting organization, caller participation and payload are required")
	}

	stored, err := this.FindByIdInOrganization(id, granter.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find the users pool")
	}

	if stored == nil {
		return nil, e.ThrowNotFound("Users pool not found in the current organization")
	}

	dao := dto.UserPoolUpdateDao{
		Name:        payload.Name,
		Description: payload.Description,
	}

	if payload.DefaultProfileId != nil {
		profile, err := this.resolveDefaultProfile(*payload.DefaultProfileId, granter, caller)

		if err != nil {
			return nil, err
		}

		dao.DefaultProfileId = &profile.ID
	}

	if payload.Metadata != nil {
		merged, err := utils.MergeJsonPatch(stored.Metadata, *payload.Metadata)

		if err != nil {
			return nil, e.ThrowBadRequest("`metadata` has to be a JSON object", utils.JSON{"field": "metadata"})
		}

		dao.Metadata = &merged
	}

	if !repo.HasChanges(dao) {
		return stored, nil
	}

	affected, err := this.userPoolRepository.Update(entity.UsersPool{ID: id}, dao)

	if err != nil {
		this.logger.Error("Failed to update a users pool", zap.String("id", id), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to update the users pool")
	}

	if affected == 0 {
		return nil, e.ThrowNotFound("Users pool not found in the current organization")
	}

	return this.FindByIdInOrganization(id, granter.ID)
}

// Track counts one signup against the month it happened in and the app it came
// through. It writes no individual event - see docs/specs/2026-09-09-tracking.md.
//
// The row lock is what makes the read-modify-write safe: `count + 1` cannot be
// expressed over a nested document with FIFO eviction, so two concurrent signups
// serialize on the row instead of losing one of the two.
func (this *UserPoolService) Track(tag tracking.Tag, payload TrackSignup) error {
	if tag != tracking.TagSignup {
		return e.ThrowInternalServerError(fmt.Sprintf("A users pool does not track `%s`", tag))
	}

	if payload.PoolId == "" || payload.AppId == "" {
		return e.ThrowInternalServerError("The pool and the app of a signup are required")
	}

	tx, err := this.txManager.Tx()

	if err != nil {
		return e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	stored, err := this.userPoolRepository.FindOneForUpdate(payload.PoolId, option)

	if err != nil {
		return e.ThrowInternalServerError("Failed to lock the users pool")
	}

	if stored == nil {
		return e.ThrowNotFound("Users pool not found")
	}

	document, err := tracking.RecordSignup(stored.Tracking, payload.AppId, payload.At)

	if err != nil {
		this.logger.Error(
			"Failed to fold a signup into the tracking of a pool",
			zap.String("pool_id", payload.PoolId),
			zap.Error(err),
		)

		return e.ThrowInternalServerError("Failed to build the tracking document")
	}

	if _, err := this.userPoolRepository.WriteTracking(payload.PoolId, document, option); err != nil {
		return e.ThrowInternalServerError("Failed to write the tracking of the users pool")
	}

	return tx.Commit()
}

// Without a profile the pool is born closed on LOGIN_PROFILE; with one, it has to
// stay inside what the granting organization itself holds.
//
// NOTE: the LOGIN_PROFILE branch is the only place in the application that names a
// profile by key. Everywhere a caller names one, it is by id.
func (this *UserPoolService) resolveDefaultProfile(
	profileId string,
	granter *entity.Organization,
	caller *entity.Participant,
) (*entity.Profile, error) {
	var profile *entity.Profile
	var err error

	if profileId == "" {
		profile, err = this.profileService.FindByKey(constants.ProfileLogin)
	} else {
		profile, err = this.profileService.FindByIdVisibleTo(profileId, granter.ID)
	}

	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, e.ThrowBadRequest(fmt.Sprintf("Profile `%s` not found", profileId))
	}

	if granter == nil || granter.Profile == nil {
		return nil, e.ThrowInternalServerError("The granting organization has no profile loaded")
	}

	if caller == nil || caller.Profile == nil {
		return nil, e.ThrowInternalServerError("The caller has no participation profile loaded")
	}

	// Against what the caller holds, not against the ceiling of its organization: a
	// member narrower than its organization must not hand out more than it has.
	ceiling, err := permissions.Resolve(granter.Profile.Permissions, caller.Profile.Permissions)

	if err != nil {
		return nil, err
	}

	within, err := permissions.IsWithin(profile.Permissions, ceiling)

	if err != nil {
		return nil, err
	}

	if !within {
		return nil, e.ThrowPermissionDeniedError(
			fmt.Sprintf(
				"The `%s` profile grants more than the current organization holds",
				profile.Key,
			),
		)
	}

	return profile, nil
}
