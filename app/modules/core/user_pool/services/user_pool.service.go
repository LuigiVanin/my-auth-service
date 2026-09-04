package services

import (
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
)

type UserPoolService struct {
	userPoolRepository upr.IUserPoolRepository
	profileService     prs.IProfileService
	cipherService      cipher.ICipherService
}

var _ IUserPoolService = &UserPoolService{}

func NewUserPoolService(
	userPoolRepository upr.IUserPoolRepository,
	profileService prs.IProfileService,
	cipherService cipher.ICipherService,
) *UserPoolService {
	return &UserPoolService{
		userPoolRepository: userPoolRepository,
		profileService:     profileService,
		cipherService:      cipherService,
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
) (*entity.UsersPool, error) {
	defaultProfile, err := this.resolveDefaultProfile(data.DefaultProfileId, granter)

	if err != nil {
		return nil, err
	}

	pool, err := this.userPoolRepository.Create(entity.UsersPool{
		Name:             data.Name,
		OwnerUserId:      data.OwnerUserId,
		OrganizationId:   data.OrganizationId,
		DefaultProfileId: defaultProfile.ID,
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

	return &dto.GetUserPoolsResponse{
		Total:  total,
		Amount: len(pools),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   pools,
	}, nil
}

// Without a profile the pool is born closed on LOGIN_PROFILE; with one, it has to
// stay inside what the granting organization itself holds.
//
// NOTE: the LOGIN_PROFILE branch is the only place in the application that names a
// profile by key. Everywhere a caller names one, it is by id.
func (this *UserPoolService) resolveDefaultProfile(
	profileId string,
	granter *entity.Organization,
) (*entity.Profile, error) {
	var profile *entity.Profile
	var err error

	if profileId == "" {
		profile, err = this.profileService.FindByKey(constants.ProfileLogin)
	} else {
		profile, err = this.profileService.FindById(profileId)
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

	within, err := permissions.IsSubsetOf(profile.Permissions, granter.Profile.Permissions)

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
