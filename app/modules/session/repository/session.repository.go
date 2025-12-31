package repository

import (
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type SessionRepository struct {
	client *gorm.DB
}

var _ ISessionRepository = &SessionRepository{}

func NewSessionRepository(client *gorm.DB) *SessionRepository {
	return &SessionRepository{
		client: client,
	}
}

func (this SessionRepository) Create(session entity.Session) (*entity.Session, error) {
	err := this.client.Create(&session).Error
	return &session, err
}

func (this SessionRepository) FindWhere(where entity.Session, with ...string) (*entity.Session, error) {
	var result entity.Session

	// NOTE: I am sorting by created_at in descending order to get the latest session ALWAYS
	whereClause := this.client.Where(where).Order("created_at desc")

	if len(with) > 0 {
		for _, relation := range with {
			whereClause = whereClause.Preload(relation)
		}
	}

	err := whereClause.First(&result).Error
	return &result, err
}

// NOTE: This is a very specific query because if a mistake is made in the parameter or implementation it can invalidate
func (this SessionRepository) BatchInvalidateAll(userId uint, appId string, currentSessionId string) error {
	return this.
		client.
		Model(&entity.Session{}).
		Where("user_id = ? AND app_id = ? AND invalidated = ? AND id != ?", userId, appId, false, currentSessionId).
		Update("invalidated", true).
		Error
}
