package services

import (
	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/participant/models"
	pr "auth_service/app/modules/core/participant/repository"
	entity "auth_service/infra/entities"
	"auth_service/shared/permissions"
	repo "auth_service/shared/repository"

	"go.uber.org/zap"
)

type ParticipantService struct {
	participantRepository pr.IParticipantRepository
	logger                *zap.Logger
}

var _ IParticipantService = &ParticipantService{}

func NewParticipantService(participantRepository pr.IParticipantRepository, logger *zap.Logger) *ParticipantService {
	return &ParticipantService{
		participantRepository: participantRepository,
		logger:                logger,
	}
}

// NOTE: does not check that the organization and the user share a pool. That
// needs both rows, and reaching for them here would make this service depend on
// the organization service, which already depends on this one. The callers
// enforce it: registration and CreateForUser build both sides in the same pool,
// Switch checks it explicitly, and a future invite endpoint has to as well.
func (this *ParticipantService) Create(
	organizationId string,
	userId uint,
	profileId string,
) (*entity.Participant, error) {
	participant, err := this.participantRepository.Create(entity.Participant{
		OrganizationId: organizationId,
		UserId:         userId,
		ProfileId:      profileId,
	})

	if err != nil {
		this.logger.Error(
			"Failed to create participant",
			zap.String("organization_id", organizationId),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)

		return nil, e.ThrowInternalServerError("Failed to create participant")
	}

	return participant, nil
}

func (this *ParticipantService) FindForUserInOrganization(
	organizationId string,
	userId uint,
) (*entity.Participant, error) {
	participant, err := this.participantRepository.FindByOrganizationAndUser(
		organizationId,
		userId,
		repo.Option{With: []string{"Profile"}},
	)

	if err != nil {
		this.logger.Error(
			"Failed to look up a participation",
			zap.String("organization_id", organizationId),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)

		return nil, e.ThrowInternalServerError("Failed to find participant")
	}

	return participant, nil
}

func (this *ParticipantService) FindAllInOrganization(
	organizationId string,
	requestingUserId uint,
	query *dto.GetParticipantsQuery,
) (*dto.GetParticipantsResponse, error) {
	requester, err := this.participantRepository.FindByOrganizationAndUser(
		organizationId,
		requestingUserId,
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find participant")
	}

	if requester == nil {
		return nil, e.ThrowNotAParticipant("User does not participate in this organization")
	}

	option := repo.Option{
		With: []string{"User", "Profile"},
		Sort: []repo.Sort{{Column: "created_at", Direction: repo.Asc}},
	}

	if query != nil {
		option = option.Paginate(query.Skip, query.Limit)
	}

	participants, err := this.participantRepository.FindByOrganizationId(organizationId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to fetch participants")
	}

	total, err := this.participantRepository.FindByOrganizationIdCount(organizationId, option)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to count participants")
	}

	return &dto.GetParticipantsResponse{
		Total:  total,
		Amount: len(participants),
		Skip:   option.Skip,
		Limit:  option.Size(),
		Data:   participants,
	}, nil
}

func (this *ParticipantService) FindForCurrentOrganization(user *entity.User) (*ResolvedParticipation, error) {
	if user == nil || user.CurrentOrganization == nil || user.CurrentOrganization.Profile == nil {
		return nil, e.ThrowInternalServerError("A user with its current organization is required")
	}

	participant, err := this.FindForUserInOrganization(user.CurrentOrganizationId, user.ID)

	if err != nil {
		return nil, err
	}

	if participant == nil {
		return nil, e.ThrowNotAParticipant("User does not participate in its current organization")
	}

	if participant.Profile == nil {
		return nil, e.ThrowInternalServerError("The participation of the user has no profile loaded")
	}

	resolved, err := permissions.Resolve(
		user.CurrentOrganization.Profile.Permissions,
		participant.Profile.Permissions,
	)

	if err != nil {
		this.logger.Error("Failed to resolve participant permissions", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to resolve permissions")
	}

	return &ResolvedParticipation{Participant: participant, Permissions: resolved}, nil
}
