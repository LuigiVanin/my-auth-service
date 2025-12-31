package repository

import (
	entity "auth_service/infra/entities"
)

type ISessionRepository interface {
	Create(session entity.Session) (*entity.Session, error)
	FindWhere(where entity.Session, with ...string) (*entity.Session, error)
	BatchInvalidateAll(userId uint, appId string, currentSessionId string) error
}
