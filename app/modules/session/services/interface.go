package services

import (
	"auth_service/app/models/dto"
	entity "auth_service/infra/entities"
)

type ISessionService interface {
	CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error)

	EncryptSessionToken(sessionId string, token string, secretKey string) (string, error)
	DecryptSessionToken(tokenString string, secretKey string) (string, string, error)
}
