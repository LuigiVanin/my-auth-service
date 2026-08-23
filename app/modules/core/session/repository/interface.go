package repository

import (
	dto "auth_service/app/modules/core/session/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
)

type ISessionRepository interface {
	FindOne(where entity.Session, options ...repo.Option) (*entity.Session, error)
	Create(session entity.Session, options ...repo.Option) (*entity.Session, error)
	Update(where entity.Session, data dto.SessionUpdateDao, options ...repo.Option) (int64, error)

	// Dedicated queries: all of these depend on `invalidated = false`, which a
	// typed `where` cannot express because gorm drops zero valued fields.
	FindActive(sessionId string, options ...repo.Option) (*entity.Session, error)
	InvalidateAllExcept(userId uint, appId string, currentSessionId string, options ...repo.Option) (int64, error)
	TouchLastUsedAt(sessionId string, options ...repo.Option) (int64, error)
}
