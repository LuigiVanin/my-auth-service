package services

import entity "auth_service/infra/entities"

type IUserService interface {
	IsAlreadyCreated(email string, app *entity.App) (bool, error)
}
