package services

import (
	dto "auth_service/app/modules/core/app/models"
	entity "auth_service/infra/entities"
)

type IAppService interface {
	CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, payload *dto.CreateAppPayload) (*entity.App, error)

	FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error)

	FindById(id string) (*entity.App, error)

	FindAllUserApps(userId string) ([]entity.App, error)

	Update(user *entity.User, app *entity.App) (*entity.App, error)
}
