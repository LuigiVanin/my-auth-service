package services

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
)

// Every method takes the current organization: it is the scope apps are created
// in, listed from and updated within.
type IAppService interface {
	CreateWithUserPool(
		currentUser *entity.User,
		currentApp *entity.App,
		currentOrganization *entity.Organization,
		participant *entity.Participant,
		payload *dto.CreateAppPayload,
	) (*entity.App, error)

	FindAll(
		currentUser *entity.User,
		currentApp *entity.App,
		currentOrganization *entity.Organization,
		query *dto.GetAppsQuery,
	) (*dto.GetAppsResponse, error)

	FindById(id string, currentOrganization *entity.Organization) (*entity.App, error)

	FindAllUserApps(userId string, currentOrganization *entity.Organization) ([]entity.App, error)

	// Update writes only the fields the payload filled, and merges Metadata into
	// what is stored rather than replacing it.
	Update(id string, currentOrganization *entity.Organization, payload *dto.UpdateApp) (*entity.App, error)
}
