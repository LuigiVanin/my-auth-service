package services

import (
	entity "auth_service/infra/entities"
	dto "auth_service/shared/models"
)

type ISessionService interface {
	CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error)

	EncryptSessionToken(sessionId string, token string, secretKey string) (string, error)
	DecryptSessionToken(tokenString string, secretKey string) (string, string, error)

	UseSession(session string) error

	// Revoke closes one session on purpose - the logout. Revoking every session of
	// a user happens inside the transaction of the email change and therefore goes
	// through the repository, not through here.
	Revoke(sessionId string) error
}
