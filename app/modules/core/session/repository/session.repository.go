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

// RevokedRefreshWindow is what a revoked session carries in refresh_expires_at.
//
// The epoch and not time.Now(): no legitimate session ends in 1970, so the row
// keeps saying it was closed on purpose rather than that it simply ran out.
// `invalidated` cannot carry that meaning - CreateNew writes it on every older
// session of the user, and /auth/refresh accepts it for exactly that reason. See
// docs/features/pendencias.md item 22 for the column that ends this.
//
// Exported so a test can build the row a revoke produces instead of keeping its
// own copy of this value.
var RevokedRefreshWindow = time.Unix(0, 0).UTC()

func revocation() dto.SessionUpdateDao {
	invalidated := true
	expired := RevokedRefreshWindow

	return dto.SessionUpdateDao{Invalidated: &invalidated, RefreshExpiresAt: &expired}
}

// Revoke closes one session: the access token dies because FindActive filters on
// `invalidated`, and the refresh token dies because Refresh checks the window
// before anything else.
//
// NOTE: the predicate is the id alone. Adding `invalidated = false` would make a
// session already superseded by a login on another device impossible to revoke,
// which is the ordinary case rather than the exception.
func (this *SessionRepository) Revoke(sessionId string, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", sessionId).
		Updates(repo.BuildUpdateMap(revocation()))

	return result.RowsAffected, result.Error
}

// RevokeAllForUser closes every session a user holds on one app. It is the logout
// that comes with changing the email: the identity moved, so nothing minted under
// the old one may survive.
func (this *SessionRepository) RevokeAllForUser(userId uint, appId string, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("user_id = ? AND app_id = ?", userId, appId).
		Updates(repo.BuildUpdateMap(revocation()))

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
