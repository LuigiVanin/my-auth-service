package repository

import (
	dto "auth_service/app/modules/core/organization/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type IOrganizationRepository interface {
	FindOne(where entity.Organization, options ...repo.Option) (*entity.Organization, error)
	Create(organization entity.Organization, options ...repo.Option) (*entity.Organization, error)
	Update(where entity.Organization, data dto.OrganizationUpdateDao, options ...repo.Option) (int64, error)

	FindByParticipantUserId(userId uint, options ...repo.Option) ([]entity.Organization, error)
	FindByParticipantUserIdCount(userId uint, options ...repo.Option) (int64, error)
}
