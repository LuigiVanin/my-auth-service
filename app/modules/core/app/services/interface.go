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

	Update(user *entity.User, app *entity.App, currentOrganization *entity.Organization) (*entity.App, error)
}
