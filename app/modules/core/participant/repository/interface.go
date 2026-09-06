package repository

import (
	dto "auth_service/app/modules/core/participant/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IParticipantRepository interface {
	Create(participant entity.Participant, options ...repo.Option) (*entity.Participant, error)
	Update(where entity.Participant, data dto.ParticipantUpdateDao, options ...repo.Option) (int64, error)
	Delete(where entity.Participant, options ...repo.Option) (int64, error)

	FindByOrganizationAndUser(organizationId string, userId uint, options ...repo.Option) (*entity.Participant, error)
	FindByOrganizationId(organizationId string, options ...repo.Option) ([]entity.Participant, error)
	FindByOrganizationIdCount(organizationId string, options ...repo.Option) (int64, error)
}
