package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type ISessionService interface {
	CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error)
}
