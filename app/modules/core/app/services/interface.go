package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type IAppService interface {
	CreateWithUserPool(currentUser *entity.User, currentApp *entity.App, payload *dto.CreateAppPayload) (*entity.App, error)
	FindAll(currentUser *entity.User, currentApp *entity.App, query *dto.GetAppsQuery) (*dto.GetAppsResponse, error)
	FindById(id string) (*entity.App, error)
	FindAllUserApps(userId string) ([]entity.App, error)
	Update(id string, app *entity.App) (*entity.App, error)
}
