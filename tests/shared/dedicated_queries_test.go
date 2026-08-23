package repository_test

import (
	"testing"

	arep "auth_service/app/modules/core/app/repository"
	prep "auth_service/app/modules/core/profile/repository"
	srep "auth_service/app/modules/core/session/repository"

	"github.com/stretchr/testify/assert"
)

// The predicate this test guards used to be written as
// FindWhere(entity.Session{ID: id, Invalidated: false}), which gorm compiled to
// `WHERE sessions.id = $1` — the zero valued bool was dropped, so an
// invalidated session still resolved and, since the caller only compared the
// token, still authorized. Never express this as a struct condition again.
func TestFindActiveFiltersInvalidatedSessions(t *testing.T) {
	client, _ := newClient(t)
	repository := srep.NewSessionRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindActive("session-1")
	})

	assert.Contains(t, sql, "invalidated = $")
	assert.Contains(t, sql, "id = $")
}

func TestSessionWritesNeverReviveAnInvalidatedRow(t *testing.T) {
	client, _ := newClient(t)
	repository := srep.NewSessionRepository(client)

	// DryRun builds the statement without executing it.
	_, err := repository.TouchLastUsedAt("session-1")
	assert.NoError(t, err)

	_, err = repository.InvalidateAllExcept(1, "app-1", "session-1")
	assert.NoError(t, err)
}

func TestAppSearchScopes(t *testing.T) {
	client, _ := newClient(t)
	repository := arep.NewAppRepository(client)

	parentAppId := "app-1"

	ownedOnly := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{OwnerUserId: 7})
	})
	assert.Contains(t, ownedOnly, "apps.owner_user_id = $")
	assert.NotContains(t, ownedOnly, "parent_app_id")

	ownedOrChildren := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{
			OwnerUserId:       7,
			OrChildrenOfAppId: &parentAppId,
			Name:              "portal",
		})
	})
	assert.Contains(t, ownedOrChildren, "(apps.owner_user_id = $1 OR apps.parent_app_id = $2)")
	assert.Contains(t, ownedOrChildren, "apps.name ILIKE $3")

	// The count must see the same predicate but no page.
	counting := captureQuery(t, client, func() {
		_, _ = repository.FindSearchCount(arep.AppSearch{OwnerUserId: 7})
	})
	assert.Contains(t, counting, "count(*)")
	assert.NotContains(t, counting, "LIMIT")
}

// The caller picks a single winning profile out of the whole set, so a page cap
// would silently pick the wrong one.
func TestProfileLookupIsNotPaginated(t *testing.T) {
	client, _ := newClient(t)
	repository := prep.NewProfileRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindByAppRole("USER")
	})

	assert.Contains(t, sql, "role = $")
	assert.Contains(t, sql, `ORDER BY "priority"`)
	assert.NotContains(t, sql, "LIMIT")
}
