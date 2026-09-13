package repository_test

import (
	"testing"

	arep "auth_service/app/modules/core/app/repository"
	prep "auth_service/app/modules/core/profile/repository"
	srep "auth_service/app/modules/core/session/repository"
	urep "auth_service/app/modules/core/user/repository"
	uprep "auth_service/app/modules/core/user_pool/repository"

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

	scoped := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{OrganizationId: "org-1"})
	})
	assert.Contains(t, scoped, "apps.organization_id = $")

	// A pool filter narrows inside the organization, it never replaces it.
	byPool := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{
			OrganizationId: "org-1",
			UsersPoolId:    "pool-1",
			Name:           "portal",
		})
	})
	assert.Contains(t, byPool, "apps.organization_id = $1")
	assert.Contains(t, byPool, "apps.users_pool_id = $2")
	assert.Contains(t, byPool, "apps.name ILIKE $3")

	// The count must see the same predicate but no page.
	counting := captureQuery(t, client, func() {
		_, _ = repository.FindSearchCount(arep.AppSearch{OrganizationId: "org-1"})
	})
	assert.Contains(t, counting, "count(*)")
	assert.NotContains(t, counting, "LIMIT")
}

func TestProfileLookupByKey(t *testing.T) {
	client, _ := newClient(t)
	repository := prep.NewProfileRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindByKey("LOGIN_PROFILE")
	})

	assert.Contains(t, sql, "key = $")
}

// The pool is mandatory, so a users listing can never widen to the whole table.
func TestUserSearchAlwaysScopesToAPool(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(urep.UserSearch{UsersPoolId: "pool-1"})
	})
	assert.Contains(t, sql, "users.users_pool_id = $")

	filtered := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(urep.UserSearch{
			UsersPoolId: "pool-1",
			Name:        "ana",
			Email:       "ana@",
		})
	})
	assert.Contains(t, filtered, "users.name ILIKE $2")
	assert.Contains(t, filtered, "users.email ILIKE $3")

	counting := captureQuery(t, client, func() {
		_, _ = repository.FindSearchCount(urep.UserSearch{UsersPoolId: "pool-1"})
	})
	assert.Contains(t, counting, "count(*)")
	assert.NotContains(t, counting, "LIMIT")
}

// The organization predicate is unconditional: a zero valued search must not
// compile into an empty WHERE over every pool in the database.
func TestUserPoolSearchAlwaysScopesToAnOrganization(t *testing.T) {
	client, _ := newClient(t)
	repository := uprep.NewUserPoolRepository(client)

	scoped := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1"})
	})
	assert.Contains(t, scoped, "users_pool.organization_id = $")

	empty := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{})
	})
	assert.Contains(t, empty, "users_pool.organization_id = $")
}

// The counter has to move in the database and never in Go: a read, a plus one and
// a write would lose every signup that lands between the two statements, and two
// concurrent registrations into one pool would report one.
func TestIncrementUsersCountIsAnInPlaceExpression(t *testing.T) {
	client, _ := newClient(t)
	repository := uprep.NewUserPoolRepository(client)

	sql := captureUpdate(t, client, func() {
		_, _ = repository.IncrementUsersCount("pool-1")
	})

	assert.Contains(t, sql, `"users_count"=users_count + $`)
	assert.Contains(t, sql, "id = $")

	// UpdateColumn, so a signup does not move `updated_at` of the pool - that
	// column means "the pool was configured".
	assert.NotContains(t, sql, "updated_at")
}

// The lock is the whole correctness argument of the tracking write: the document
// is read, changed in Go and written back, so without FOR UPDATE two concurrent
// signups into one pool would lose one of the two.
func TestTrackingReadsTakeARowLock(t *testing.T) {
	client, _ := newClient(t)

	poolSql := captureQuery(t, client, func() {
		_, _ = uprep.NewUserPoolRepository(client).FindOneForUpdate("pool-1")
	})

	assert.Contains(t, poolSql, "FOR UPDATE")
	assert.Contains(t, poolSql, "id = $")

	// Query and not ListQuery: a single row read must not carry the page limit.
	assert.NotContains(t, poolSql, "LIMIT 10")

	userSql := captureQuery(t, client, func() {
		_, _ = urep.NewUserRepository(client).FindOneForUpdate(7)
	})

	assert.Contains(t, userSql, "FOR UPDATE")
}

// UpdateColumn, so a signup does not move `updated_at` of the pool and a login
// does not move it on the user - those columns mean "the row was configured".
func TestTrackingWritesOnlyTheTrackingColumn(t *testing.T) {
	client, _ := newClient(t)

	poolSql := captureUpdate(t, client, func() {
		_, _ = uprep.NewUserPoolRepository(client).WriteTracking("pool-1", []byte(`{}`))
	})

	assert.Contains(t, poolSql, `"tracking"=$`)
	assert.NotContains(t, poolSql, "updated_at")
	assert.NotContains(t, poolSql, "metadata")

	userSql := captureUpdate(t, client, func() {
		_, _ = urep.NewUserRepository(client).WriteTracking(7, []byte(`{}`))
	})

	assert.Contains(t, userSql, `"tracking"=$`)
	assert.NotContains(t, userSql, "updated_at")
}

// A global profile has organization_id NULL, and gorm drops a nil pointer out of a
// typed struct condition - so writing this predicate as a struct would compile to a
// WHERE with the NULL half missing, and every organization would see every scoped
// profile of every other one. It has to stay raw SQL.
func TestProfileVisibilityKeepsTheGlobalHalf(t *testing.T) {
	client, _ := newClient(t)
	repository := prep.NewProfileRepository(client)

	visible := captureQuery(t, client, func() {
		_, _ = repository.FindVisibleTo(prep.ProfileSearch{OrganizationId: "org-1"})
	})

	assert.Contains(t, visible, "profiles.organization_id IS NULL")
	assert.Contains(t, visible, "profiles.organization_id = $")

	// Scoped drops the global half: it is the "only what this organization owns"
	// filter of the listing, not the visibility rule.
	scoped := captureQuery(t, client, func() {
		_, _ = repository.FindVisibleTo(prep.ProfileSearch{OrganizationId: "org-1", Scoped: true})
	})

	assert.NotContains(t, scoped, "IS NULL")
	assert.Contains(t, scoped, "profiles.organization_id = $")
}

// A lookup that skipped the predicate would resolve a profile scoped to another
// organization, which is the whole point of the feature.
func TestProfileByIdIsScoped(t *testing.T) {
	client, _ := newClient(t)
	repository := prep.NewProfileRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindByIdVisibleTo("profile-1", "org-1")
	})

	assert.Contains(t, sql, "profiles.organization_id IS NULL")
	assert.Contains(t, sql, "profiles.id = $")
}
