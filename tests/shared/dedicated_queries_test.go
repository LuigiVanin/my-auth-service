package repository_test

import (
	"testing"
	"time"

	arep "auth_service/app/modules/core/app/repository"
	idto "auth_service/app/modules/core/invite/models"
	irep "auth_service/app/modules/core/invite/repository"
	orep "auth_service/app/modules/core/organization/repository"
	prep "auth_service/app/modules/core/profile/repository"
	srep "auth_service/app/modules/core/session/repository"
	urep "auth_service/app/modules/core/user/repository"
	uprep "auth_service/app/modules/core/user_pool/repository"
	repo "auth_service/shared/repository"

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

// Revoking has to close both windows in one statement. `invalidated` alone is not
// enough: /auth/refresh accepts an invalidated session on purpose, because
// CreateNew writes that flag on every older session of the user.
func TestRevokeClosesBothTokenWindows(t *testing.T) {
	client, _ := newClient(t)
	repository := srep.NewSessionRepository(client)

	sql := captureUpdate(t, client, func() {
		_, _ = repository.Revoke("session-1")
	})

	assert.Contains(t, sql, "invalidated")
	assert.Contains(t, sql, "refresh_expires_at")
	assert.Contains(t, sql, "updated_at")
	assert.Contains(t, sql, "id = $")

	// The id alone: a session already superseded by a login on another device is
	// the ordinary thing to revoke, not the exception.
	assert.NotContains(t, sql, "invalidated = $")
}

func TestRevokeAllForUserIsScopedToTheUserAndTheApp(t *testing.T) {
	client, _ := newClient(t)
	repository := srep.NewSessionRepository(client)

	sql := captureUpdate(t, client, func() {
		_, _ = repository.RevokeAllForUser(1, "app-1")
	})

	assert.Contains(t, sql, "refresh_expires_at")
	assert.Contains(t, sql, "user_id = $")
	assert.Contains(t, sql, "app_id = $")
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

// The OR has to stay inside parentheses, or it would match users of other pools.
func TestUserSearchMatchesNameOrEmailInsideThePool(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(urep.UserSearch{UsersPoolId: "pool-1", Search: "ana"})
	})

	assert.Contains(t, sql, "users.users_pool_id = $1")
	assert.Contains(t, sql, "(users.name ILIKE $2 OR users.email ILIKE $3)")
	assert.NotContains(t, sql, "JOIN")
}

func TestUserSearchFiltersByProfileFlagsAndSignupWindow(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	no := false
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(urep.UserSearch{
			UsersPoolId:      "pool-1",
			ProfileId:        "profile-1",
			EmailVerified:    &no,
			TwoFactorEnabled: &no,
			SignedUpAfter:    &from,
			SignedUpBefore:   &before,
		})
	})

	assert.Contains(t, sql, "users.current_organization_id IN (SELECT organizations.id FROM organizations WHERE organizations.profile_id = $2)")
	// false has to reach the statement: a struct condition would have dropped it.
	assert.Contains(t, sql, "users.verify_email = $3")
	assert.Contains(t, sql, "users.two_factor_enabled = $4")
	assert.Contains(t, sql, "users.created_at >= $5")
	assert.Contains(t, sql, "users.created_at < $6")

	counting := captureQuery(t, client, func() {
		_, _ = repository.FindSearchCount(urep.UserSearch{UsersPoolId: "pool-1", EmailVerified: &no})
	})
	assert.Contains(t, counting, "users.verify_email = $2")

	bare := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(urep.UserSearch{UsersPoolId: "pool-1"})
	})
	assert.NotContains(t, bare, "verify_email")
	assert.NotContains(t, bare, "organizations.profile_id")
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

// avatar_key is absent from the three update daos, so this is its only writer.
// Update and not UpdateColumn: changing the avatar moves `updated_at`. A nil key
// has to reach the statement, or removing an avatar would write nothing.
func TestAvatarKeyWritesOnlyItsColumn(t *testing.T) {
	client, _ := newClient(t)
	key := "avatar/users/7/9f1c3a7e2b4d6081.webp"

	for name, write := range map[string]func(){
		"users":         func() { _, _ = urep.NewUserRepository(client).WriteAvatarKey(7, &key) },
		"organizations": func() { _, _ = orep.NewOrganizationRepository(client).WriteAvatarKey("org-1", &key) },
		"apps":          func() { _, _ = arep.NewAppRepository(client).WriteAvatarKey("app-1", nil) },
	} {
		sql := captureUpdate(t, client, write)

		assert.Contains(t, sql, `UPDATE "`+name+`" SET "avatar_key"=$1`, name)
		assert.Contains(t, sql, `"updated_at"=$2`, name)
		assert.Contains(t, sql, "WHERE id = $3", name)
		assert.NotContains(t, sql, "metadata", name)
	}
}

const invitePending = "accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()"

// Both transitions match a pending row only: two accepts at once end with one
// participation, and an accept never revives a revoked invite. Written as raw SQL
// because a struct condition would drop the nil pointers in silence.
func TestInviteTransitionsOnlyMoveAPendingRow(t *testing.T) {
	client, _ := newClient(t)
	repository := irep.NewInviteRepository(client)

	for column, write := range map[string]func(){
		"accepted_at": func() { _, _ = repository.Accept("invite-1", time.Now()) },
		"revoked_at":  func() { _, _ = repository.Revoke("invite-1", time.Now()) },
	} {
		sql := captureUpdate(t, client, write)

		assert.Contains(t, sql, `UPDATE "invites" SET "`+column+`"=$1`, column)
		assert.Contains(t, sql, `"updated_at"=$2`, column)
		assert.Contains(t, sql, "id = $3", column)
		assert.Contains(t, sql, invitePending, column)
	}
}

func TestInviteListingFiltersTheStatusInSql(t *testing.T) {
	client, _ := newClient(t)
	repository := irep.NewInviteRepository(client)

	for status, predicate := range map[idto.InviteStatus]string{
		idto.InviteStatusPending:  invitePending,
		idto.InviteStatusAccepted: "accepted_at IS NOT NULL",
		idto.InviteStatusRevoked:  "revoked_at IS NOT NULL",
		idto.InviteStatusExpired:  "expires_at <= now()",
	} {
		listing := captureQuery(t, client, func() {
			_, _ = repository.FindForOrganization("org-1", status)
		})

		assert.Contains(t, listing, "organization_id = $", status)
		assert.Contains(t, listing, predicate, status)
		assert.Contains(t, listing, `ORDER BY "created_at" DESC`, status)
		assert.Contains(t, listing, "LIMIT", status)

		count := captureQuery(t, client, func() {
			_, _ = repository.FindForOrganizationCount("org-1", status)
		})

		assert.Contains(t, count, predicate, status)
		assert.NotContains(t, count, "LIMIT", status)
	}

	everything := captureQuery(t, client, func() {
		_, _ = repository.FindForOrganization("org-1", "")
	})

	assert.NotContains(t, everything, "accepted_at")

	_, err := repository.FindForOrganization("org-1", "lost")
	assert.Error(t, err)
}

// Every pending invite for the address is revoked when a new one replaces them,
// so the lookup is not paginated - a page would leave the rest alive.
func TestInvitePendingForAnAddressIsUnpaginated(t *testing.T) {
	client, _ := newClient(t)
	repository := irep.NewInviteRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindPendingFor("org-1", "guest@example.com")
	})

	assert.Contains(t, sql, "organization_id = $1 AND email = $2")
	assert.Contains(t, sql, invitePending)
	assert.NotContains(t, sql, "LIMIT")
}

func TestUserPoolSearchFiltersByCeilingAndScope(t *testing.T) {
	client, _ := newClient(t)
	repository := uprep.NewUserPoolRepository(client)

	closed := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1", Ceiling: "closed"})
	})
	assert.Contains(t, closed, "users_pool.organization_id = $1")
	assert.Contains(t, closed, "users_pool.default_profile_id IN (SELECT profiles.id FROM profiles WHERE profiles.key = $2)")

	// scoped is "not closed and not broad": the NOT must wrap the whole COALESCE.
	scoped := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1", Ceiling: "scoped"})
	})
	assert.Contains(t, scoped, "profiles.key <> $2 AND NOT COALESCE(")
	assert.Contains(t, scoped, "grant_key LIKE '%*%'")

	broad := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1", Ceiling: "broad"})
	})
	assert.Contains(t, broad, "profiles.key <> $2 AND COALESCE(")

	system := captureQuery(t, client, func() {
		_, _ = repository.FindSearchCount(uprep.UserPoolSearch{OrganizationId: "org-1", Scope: "system"})
	})
	assert.Contains(t, system, "profiles.organization_id IS NULL")

	organization := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1", Scope: "organization"})
	})
	assert.Contains(t, organization, "profiles.organization_id = $2")

	// An unknown value never compiles into a predicate - and never widens the scope.
	unknown := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1", Ceiling: "everything"})
	})
	assert.NotContains(t, unknown, "default_profile_id")
	assert.Contains(t, unknown, "users_pool.organization_id = $1")
}

func TestAppSearchFiltersByLoginTypeAndVisibility(t *testing.T) {
	client, _ := newClient(t)
	repository := arep.NewAppRepository(client)

	public := false
	sql := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{
			OrganizationId: "org-1",
			LoginType:      "WITH_LOGIN",
			Private:        &public,
		})
	})
	assert.Contains(t, sql, "apps.organization_id = $1")
	// Text, not the enum: auth_method has no WITH_LOGIN.
	assert.Contains(t, sql, "$2 = ANY(apps.login_types::text[])")
	assert.Contains(t, sql, "apps.private = $3")

	bare := captureQuery(t, client, func() {
		_, _ = repository.FindSearch(arep.AppSearch{OrganizationId: "org-1"})
	})
	assert.NotContains(t, bare, "login_types")
	assert.NotContains(t, bare, "apps.private")
}

func TestListOrderIsTotal(t *testing.T) {
	client, _ := newClient(t)
	pools := uprep.NewUserPoolRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = pools.FindSearch(uprep.UserPoolSearch{OrganizationId: "org-1"}, repo.Option{
			Sort: []repo.Sort{{Column: "users_count", Direction: repo.Desc}, {Column: "id", Direction: repo.Asc}},
		})
	})
	assert.Contains(t, sql, `ORDER BY "users_count" DESC,"id"`)

	counting := captureQuery(t, client, func() {
		_, _ = pools.FindSearchCount(uprep.UserPoolSearch{OrganizationId: "org-1"}, repo.Option{
			Sort: []repo.Sort{{Column: "users_count", Direction: repo.Desc}},
		})
	})
	assert.NotContains(t, counting, "ORDER BY")
}
