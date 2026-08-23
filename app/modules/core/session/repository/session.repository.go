package repository

import (
	"time"

	dto "auth_service/app/modules/core/session/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type SessionRepository struct {
	repo.BaseRepository[entity.Session, dto.SessionUpdateDao]
}

var _ ISessionRepository = &SessionRepository{}

func NewSessionRepository(client *gorm.DB) *SessionRepository {
	return &SessionRepository{
		BaseRepository: repo.NewBaseRepository[entity.Session, dto.SessionUpdateDao](client),
	}
}

// FindActive returns a session only while it is still valid.
//
// NOTE: this used to be FindWhere(entity.Session{ID: id, Invalidated: false}),
// which silently matched invalidated sessions too: gorm drops zero valued
// fields from a struct condition, so `Invalidated: false` produced no SQL at
// all. The predicate has to be explicit.
func (this *SessionRepository) FindActive(sessionId string, options ...repo.Option) (*entity.Session, error) {
	query := this.ListQuery(options...).
		Where("id = ? AND invalidated = ?", sessionId, false)

	return repo.FirstOrNil[entity.Session](query)
}

// InvalidateAllExcept closes every other session of the user on the app.
//
// NOTE: a mistake in this predicate logs people out, so it stays a dedicated
// method with named parameters instead of a condition assembled by a caller.
func (this *SessionRepository) InvalidateAllExcept(
	userId uint,
	appId string,
	currentSessionId string,
	options ...repo.Option,
) (int64, error) {
	invalidated := true

	result := this.Query(options...).
		Where(
			"user_id = ? AND app_id = ? AND invalidated = ? AND id != ?",
			userId, appId, false, currentSessionId,
		).
		Updates(repo.BuildUpdateMap(dto.SessionUpdateDao{Invalidated: &invalidated}))

	return result.RowsAffected, result.Error
}

// TouchLastUsedAt stamps the session, refusing to revive an invalidated one.
func (this *SessionRepository) TouchLastUsedAt(sessionId string, options ...repo.Option) (int64, error) {
	now := time.Now()

	result := this.Query(options...).
		Where("id = ? AND invalidated = ?", sessionId, false).
		Updates(repo.BuildUpdateMap(dto.SessionUpdateDao{LastUsedAt: &now}))

	return result.RowsAffected, result.Error
}
