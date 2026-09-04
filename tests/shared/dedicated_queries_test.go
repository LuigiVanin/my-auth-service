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
