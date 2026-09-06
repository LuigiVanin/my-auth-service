package services

import (
	"encoding/json"
	"fmt"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/organization/models"
	or "auth_service/app/modules/core/organization/repository"
	pdto "auth_service/app/modules/core/participant/models"
	prep "auth_service/app/modules/core/participant/repository"
	ps "auth_service/app/modules/core/participant/services"
	prr "auth_service/app/modules/core/profile/repository"
	prs "auth_service/app/modules/core/profile/services"
	udto "auth_service/app/modules/core/user/models"
	us "auth_service/app/modules/core/user/services"
	ups "auth_service/app/modules/core/user_pool/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"

	"go.uber.org/zap"
)

type OrganizationService struct {
	organizationRepository or.IOrganizationRepository

	// The write of a participation inside a unit of work goes to the repository,
	// so it lands on the same repo.Option as everything else in the transaction.
	// The service is kept for the reads, which carry the resolving rule.
	participantRepository prep.IParticipantRepository
	participantService    ps.IParticipantService

	// The Admin profile of a new organization is written inside the transaction that
	// creates it, so it goes to the repository too. Reads stay on the service.
	profileRepository prr.IProfileRepository
	profileService    prs.IProfileService
	userPoolService   ups.IUserPoolService
	userService       us.IUserService
	txManager         repo.ITransactionManager
	logger            *zap.Logger
}

var _ IOrganizationService = &OrganizationService{}

func NewOrganizationService(
	organizationRepository or.IOrganizationRepository,
	participantRepository prep.IParticipantRepository,
	participantService ps.IParticipantService,
	profileRepository prr.IProfileRepository,
	profileService prs.IProfileService,
	userPoolService ups.IUserPoolService,
	userService us.IUserService,
	txManager repo.ITransactionManager,
	logger *zap.Logger,
) *OrganizationService {
	return &OrganizationService{
		organizationRepository: organizationRepository,
		participantRepository:  participantRepository,
		participantService:     participantService,
		profileRepository:      profileRepository,
		profileService:         profileService,
		userPoolService:        userPoolService,
		userService:            userService,
		txManager:              txManager,
		logger:                 logger,
	}
}

func (this *OrganizationService) Create(data dto.CreateOrganizationData) (*entity.Organization, error) {
	metadata := data.Metadata

	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	organization, err := this.organizationRepository.Create(entity.Organization{
		UsersPoolId: data.UsersPoolId,
		ProfileId:   data.ProfileId,
		OwnerUserId: data.OwnerUserId,
		Name:        data.Name,
		Description: data.Description,
		Metadata:    metadata,
	})

	if err != nil {
		this.logger.Error("Failed to create organization", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create organization")
	}

	return organization, nil
}

func (this *OrganizationService) SetOwner(organizationId string, ownerUserId uint) error {
	affected, err := this.organizationRepository.Update(
		entity.Organization{ID: organizationId},
		dto.OrganizationUpdateDao{OwnerUserId: &ownerUserId},
	)

	if err != nil {
		this.logger.Error("Failed to set the organization owner", zap.Error(err))
		return e.ThrowInternalServerError("Failed to set the organization owner")
	}

	if affected == 0 {
		return e.ThrowInternalServerError("Organization disappeared before its owner could be set")
	}

	return nil
}

func (this *OrganizationService) FindById(id string) (*entity.Organization, error) {
	organization, err := this.organizationRepository.FindOne(
		entity.Organization{ID: id},
		repo.Option{With: []string{"Profile"}},
	)

	if err != nil {
		this.logger.Error("Failed to find organization", zap.String("organization_id", id), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find organization")
	}

	return organization, nil
}

func (this *OrganizationService) FindAllForUser(
	userId uint,
	query *dto.GetOrganizationsQuery,
) (*dto.GetOrganizationsResponse, error) {
	option := repo.Option{
		With: []string{"Profile"},
		Sort: []repo.Sort{{Column: "created_at", Direction: repo.Asc}},
	}

	if query != nil {
		option = option.Paginate(query.Skip, query.Limit)
	}

	organizations, err := this.organizationRepository.FindByParticipantUserId(userId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch organizations")
	}

	total, err := this.organizationRepository.FindByParticipantUserIdCount(userId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count organizations")
	}

	return &dto.GetOrganizationsResponse{
		Total:  total,
		Amount: len(organizations),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   organizations,
	}, nil
}

// NOTE: a missing organization, one in another pool and one the user is not part
// of all answer the same NOT_A_PARTICIPANT. Telling them apart would turn this
// into an oracle for organization ids.
func (this *OrganizationService) Switch(
	user *entity.User,
	organizationId string,
) (*entity.Organization, error) {
	if user == nil {
		return nil, e.ThrowInternalServerError("Current user is required")
	}

	organization, err := this.FindById(organizationId)

	if err != nil {
		return nil, err
	}

	if organization == nil || organization.UsersPoolId != user.UsersPoolId {
		return nil, e.ThrowNotAParticipant("User does not participate in this organization")
	}

	participant, err := this.participantService.FindForUserInOrganization(organizationId, user.ID)

	if err != nil {
		return nil, err
	}

	if participant == nil {
		return nil, e.ThrowNotAParticipant("User does not participate in this organization")
	}

	if _, err := this.userService.Update(
		entity.User{ID: user.ID},
		udto.UserUpdateDao{CurrentOrganizationId: &organizationId},
	); err != nil {
		this.logger.Error("Failed to switch the current organization", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to switch the current organization")
	}

	return organization, nil
}

// The current organization of the caller is deliberately left alone: creating is
// not entering, the client calls switch when it wants to move.
func (this *OrganizationService) CreateForUser(
	currentUser *entity.User,
	currentOrganization *entity.Organization,
	payload *dto.CreateOrganizationPayload,
) (*entity.Organization, error) {
	if currentUser == nil || currentOrganization == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current user, current organization and payload are required")
	}

	ceiling, err := this.resolveCeiling(currentUser.UsersPoolId, currentOrganization, payload.ProfileId)

	if err != nil {
		return nil, err
	}

	tx, err := this.txManager.Tx()

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	metadata := payload.Metadata

	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	organization, err := this.organizationRepository.Create(entity.Organization{
		UsersPoolId: currentUser.UsersPoolId,
		ProfileId:   ceiling.ID,
		OwnerUserId: &currentUser.ID,
		Name:        payload.Name,
		Description: payload.Description,
		Metadata:    metadata,
	}, option)

	if err != nil {
		this.logger.Error("Failed to create organization", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create organization")
	}

	admin, err := this.profileRepository.Create(prs.OrganizationAdminProfile(organization.ID), option)

	if err != nil {
		this.logger.Error("Failed to create the organization admin profile", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create the organization admin profile")
	}

	// The owner holds the ceiling of the organization it just created, no more: the
	// Admin profile is a wildcard, and a wildcard resolves back onto the ceiling
	// instead of widening it - now and after the ceiling changes.
	if _, err := this.participantRepository.Create(entity.Participant{
		OrganizationId: organization.ID,
		UserId:         currentUser.ID,
		ProfileId:      admin.ID,
	}, option); err != nil {
		this.logger.Error("Failed to create participant", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create participant")
	}

	if err := tx.Commit(); err != nil {
		return nil, e.ThrowInternalServerError("Failed to commit transaction")
	}

	organization.Profile = ceiling

	return organization, nil
}

// UpdateParticipant moves one participation onto another profile. Three refusals
// stand between it and self elevation, and none of them is implied by the others:
//
//  1. the caller does not move itself, or a wider profile it may hand out becomes a
//     wider profile it holds;
//  2. the participation of the owner is frozen, so an organization always keeps one
//     administrator nobody can demote - and nobody can lock everyone out of it;
//  3. the chosen profile has to fit inside what the caller itself holds here, which
//     is what stops two narrow participants from promoting each other.
//
// The third is the same comparison resolveCeiling and resolveDefaultProfile make,
// in the third place a profile is chosen by name.
func (this *OrganizationService) UpdateParticipant(
	organizationId string,
	participantId string,
	currentOrganization *entity.Organization,
	caller *entity.Participant,
	payload *pdto.UpdateParticipant,
) (*entity.Participant, error) {
	if currentOrganization == nil || caller == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Current organization, caller participation and payload are required")
	}

	if currentOrganization.Profile == nil || caller.Profile == nil {
		return nil, e.ThrowInternalServerError("The current organization or the caller has no profile loaded")
	}

	// The PermissionsGuard answered this request against the organization the caller
	// is currently in, so that is the only organization the write may land on.
	// Reaching another one would authorize the write with powers held elsewhere.
	if organizationId != currentOrganization.ID {
		return nil, e.ThrowPermissionDeniedError(
			"A participation can only be changed in the organization you are currently in",
		)
	}

	target, err := this.participantRepository.FindOne(
		entity.Participant{ID: participantId, OrganizationId: organizationId},
		repo.Option{With: []string{"User", "Profile"}},
	)

	if err != nil {
		this.logger.Error("Failed to look up a participation", zap.String("participant_id", participantId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find participant")
	}

	if target == nil {
		return nil, e.ThrowNotFound(fmt.Sprintf("Participant `%s` not found in this organization", participantId))
	}

	if target.ID == caller.ID {
		return nil, e.ThrowPermissionDeniedError("You cannot change the profile you participate with")
	}

	if currentOrganization.OwnerUserId != nil && target.UserId == *currentOrganization.OwnerUserId {
		return nil, e.ThrowPermissionDeniedError("The participation of the owner of an organization cannot be changed")
	}

	profile, err := this.profileService.FindByIdVisibleTo(payload.ProfileId, organizationId)

	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, e.ThrowBadRequest(fmt.Sprintf("Profile `%s` not found", payload.ProfileId))
	}

	ceiling, err := permissions.Resolve(currentOrganization.Profile.Permissions, caller.Profile.Permissions)

	if err != nil {
		this.logger.Error("Failed to resolve the ceiling of the caller", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to resolve permissions")
	}

	within, err := permissions.IsWithin(profile.Permissions, ceiling)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to compare permissions")
	}

	if !within {
		return nil, e.ThrowPermissionDeniedError(
			fmt.Sprintf("The `%s` profile grants more than you hold in this organization", profile.Key),
		)
	}

	affected, err := this.participantRepository.Update(
		entity.Participant{ID: participantId},
		pdto.ParticipantUpdateDao{ProfileId: &profile.ID},
	)

	if err != nil {
		this.logger.Error("Failed to update a participation", zap.String("participant_id", participantId), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to update participant")
	}

	if affected == 0 {
		return nil, e.ThrowNotFound(fmt.Sprintf("Participant `%s` not found in this organization", participantId))
	}

	target.ProfileId = profile.ID
	target.Profile = profile

	return target, nil
}

// A requested key is only accepted when it stays inside the ceiling of the
// organization granting it - "a layer only grants what it holds", at write time.
func (this *OrganizationService) resolveCeiling(
	usersPoolId string,
	currentOrganization *entity.Organization,
	profileId string,
) (*entity.Profile, error) {
	if profileId == "" {
		pool, err := this.userPoolService.FindById(usersPoolId)

		if err != nil {
			return nil, err
		}

		if pool == nil || pool.DefaultProfile == nil {
			return nil, e.ThrowInternalServerError("The users pool has no default profile")
		}

		return pool.DefaultProfile, nil
	}

	requested, err := this.profileService.FindByIdVisibleTo(profileId, currentOrganization.ID)

	if err != nil {
		return nil, err
	}

	if requested == nil {
		return nil, e.ThrowBadRequest(fmt.Sprintf("Profile `%s` not found", profileId))
	}

	if currentOrganization.Profile == nil {
		return nil, e.ThrowInternalServerError("Current organization has no profile loaded")
	}

	within, err := permissions.IsSubsetOf(
		requested.Permissions,
		currentOrganization.Profile.Permissions,
	)

	if err != nil {
		return nil, err
	}

	if !within {
		return nil, e.ThrowPermissionDeniedError(
			fmt.Sprintf(
				"The `%s` profile grants more than the current organization holds",
				requested.Key,
			),
		)
	}

	return requested, nil
}
