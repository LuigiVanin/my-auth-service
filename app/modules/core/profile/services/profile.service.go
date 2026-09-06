package services

import (
	"encoding/json"
	"fmt"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/profile/models"
	"auth_service/app/modules/core/profile/repository"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"
	"auth_service/shared/utils"

	"go.uber.org/zap"
)

type ProfileService struct {
	profileRepository repository.IProfileRepository
	logger            *zap.Logger
}

var _ IProfileService = &ProfileService{}

func NewProfileService(profileRepository repository.IProfileRepository, logger *zap.Logger) *ProfileService {
	return &ProfileService{
		profileRepository: profileRepository,
		logger:            logger,
	}
}

func (this *ProfileService) FindByIdVisibleTo(id string, organizationId string) (*entity.Profile, error) {
	profile, err := this.profileRepository.FindByIdVisibleTo(id, organizationId)

	if err != nil {
		this.logger.Error("Failed to look up a profile by id", zap.String("id", id), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	return profile, nil
}

func (this *ProfileService) FindByKey(key string) (*entity.Profile, error) {
	profile, err := this.profileRepository.FindByKey(key)

	if err != nil {
		this.logger.Error("Failed to look up a profile by key", zap.String("key", key), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	return profile, nil
}

func (this *ProfileService) FindAllVisibleTo(
	organizationId string,
	query *dto.GetProfilesQuery,
) (*dto.GetProfilesResponse, error) {

	if organizationId == "" {
		return nil, e.ThrowInternalServerError("Current organization is required")
	}

	search := repository.ProfileSearch{OrganizationId: organizationId}
	option := repo.Option{}

	if query != nil {
		search.Name = query.Name

		if query.OrganizationId != "" {
			search.Scoped = true
			search.OrganizationId = query.OrganizationId
		}

		option = option.Paginate(query.Skip, query.Limit)
	}

	if search.Scoped && search.OrganizationId != organizationId {
		return &dto.GetProfilesResponse{Skip: option.Skip, Limit: option.Size(), Data: []entity.Profile{}}, nil
	}

	profiles, err := this.profileRepository.FindVisibleTo(search, option)

	if err != nil {
		this.logger.Error("Failed to list profiles", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to fetch profiles")
	}

	total, err := this.profileRepository.FindVisibleToCount(search, option)

	if err != nil {
		this.logger.Error("Failed to count profiles", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to count profiles")
	}

	return &dto.GetProfilesResponse{
		Total:  total,
		Amount: len(profiles),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   profiles,
	}, nil
}

func (this *ProfileService) CreateForOrganization(
	caller *ProfileWriteContext,
	payload *dto.CreateProfile,
) (*entity.Profile, error) {

	if caller == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Caller context and payload are required")
	}

	document, err := this.documentWithin(caller, payload.Permissions.Grants)

	if err != nil {
		return nil, err
	}

	permissionDocument, err := this.marshalDocument(document)

	if err != nil {
		return nil, err
	}

	key, err := this.scopedKey(caller.Organization.ID, payload.Name)

	if err != nil {
		return nil, err
	}

	created, err := this.profileRepository.Create(entity.Profile{
		Name:           payload.Name,
		Key:            key,
		OrganizationId: &caller.Organization.ID,
		Permissions:    permissionDocument,
		Metadata:       json.RawMessage(`{}`),
	})

	if err != nil {
		this.logger.Error("Failed to create a profile", zap.String("key", key), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create profile")
	}

	return created, nil
}

func (this *ProfileService) UpdateForOrganization(
	id string,
	caller *ProfileWriteContext,
	payload *dto.UpdateProfile,
) (*entity.Profile, error) {

	if caller == nil || payload == nil {
		return nil, e.ThrowInternalServerError("Caller context and payload are required")
	}

	current, err := this.FindByIdVisibleTo(id, caller.Organization.ID)

	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, e.ThrowNotFound(fmt.Sprintf("Profile `%s` not found", id))
	}

	if err := this.writable(current, caller); err != nil {
		return nil, err
	}

	dao := dto.ProfileUpdateDao{Name: payload.Name}

	if payload.Permissions != nil {
		document, err := this.documentWithin(caller, payload.Permissions.Grants)

		if err != nil {
			return nil, err
		}

		// The endpoint only speaks grants, so an api half written by hand on the row
		// is carried over as it stands. Writing the payload whole would erase it in
		// silence, and it is exactly the granularity no grant can express.
		stored, err := permissions.Parse(current.Permissions)

		if err != nil {
			this.logger.Error("Failed to parse the stored permission document", zap.String("id", id), zap.Error(err))
			return nil, e.ThrowInternalServerError("Failed to read the stored permissions")
		}

		document.Api = stored.Api

		permissionDocument, err := this.marshalDocument(document)

		if err != nil {
			return nil, err
		}

		dao.Permissions = &permissionDocument
	}

	affected, err := this.profileRepository.Update(entity.Profile{ID: id}, dao)

	if err != nil {
		this.logger.Error("Failed to update a profile", zap.String("id", id), zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to update profile")
	}

	if affected == 0 {
		return nil, e.ThrowNotFound(fmt.Sprintf("Profile `%s` not found", id))
	}

	return this.FindByIdVisibleTo(id, caller.Organization.ID)
}

// Narrowing either through the API would be a one way door: a global row is seed, and
// the Admin row is what keeps an organization's owner tracking its ceiling.
//
// The third refusal is the one that closes self elevation: documentWithin caps a
// write at what the caller holds, but the caller editing the very row it
// participates with would still be widening itself, one accepted request at a time.
func (this *ProfileService) writable(profile *entity.Profile, caller *ProfileWriteContext) error {
	if profile.OrganizationId == nil {
		return e.ThrowPermissionDeniedError("A global profile cannot be edited")
	}

	// Built by the same function that writes it, so the two cannot drift.
	if profile.Key == ScopedKey(*profile.OrganizationId, constants.ProfileOrganizationAdmin) {
		return e.ThrowPermissionDeniedError("The Admin profile of an organization cannot be edited")
	}

	if caller.Participant != nil && profile.ID == caller.Participant.ProfileId {
		return e.ThrowPermissionDeniedError("The profile you participate with cannot be edited by you")
	}

	return nil
}

// The comparison is against the resolved Api and never against Resolved.Grants: a
// ceiling written in api declares no grant at all, so comparing lists would leave the
// platform admin unable to author anything.
func (this *ProfileService) documentWithin(
	caller *ProfileWriteContext,
	grants []string,
) (permissions.Document, error) {

	empty := permissions.Document{}

	if caller.Organization == nil || caller.Organization.Profile == nil ||
		caller.Participant == nil || caller.Participant.Profile == nil {
		return empty, e.ThrowInternalServerError("Caller has no organization or participation profile loaded")
	}

	if err := permissions.ValidateGrants(grants); err != nil {
		return empty, e.ThrowBadRequest(err.Error())
	}

	requested := permissions.Document{Grants: grants}

	document, err := this.marshalDocument(requested)

	if err != nil {
		return empty, err
	}

	ceiling, err := permissions.Resolve(
		caller.Organization.Profile.Permissions,
		caller.Participant.Profile.Permissions,
	)

	if err != nil {
		this.logger.Error("Failed to resolve the ceiling of the caller", zap.Error(err))
		return empty, e.ThrowInternalServerError("Failed to resolve permissions")
	}

	within, err := permissions.IsWithin(document, ceiling)

	if err != nil {
		return empty, e.ThrowInternalServerError("Failed to compare permissions")
	}

	if !within {
		return empty, e.ThrowPermissionDeniedError(
			"The requested permissions exceed what you hold in this organization",
			utils.JSON{"requested": grants, "granted": ceiling.Grants},
		)
	}

	return requested, nil
}

func (this *ProfileService) marshalDocument(document permissions.Document) (json.RawMessage, error) {
	encoded, err := json.Marshal(document)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to build the permission document")
	}

	return encoded, nil
}

func (this *ProfileService) scopedKey(organizationId string, name string) (string, error) {
	key := ScopedKey(organizationId, name)

	if key == "" {
		return "", e.ThrowBadRequest(fmt.Sprintf("`%s` has no character a profile key can be built from", name))
	}

	return key, nil
}
