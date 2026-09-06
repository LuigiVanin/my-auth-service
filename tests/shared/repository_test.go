package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	odto "auth_service/app/modules/core/otp/models"
	orep "auth_service/app/modules/core/otp/repository"
	udto "auth_service/app/modules/core/user/models"
	urep "auth_service/app/modules/core/user/repository"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- fake conn pool, so the SQL can be inspected without a database ---

type fakeTx struct {
	commits   int
	rollbacks int
}

func (this *fakeTx) PrepareContext(c context.Context, q string) (*sql.Stmt, error) { return nil, nil }
func (this *fakeTx) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return nil, nil
}
func (this *fakeTx) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return nil, nil
}
func (this *fakeTx) QueryRowContext(c context.Context, q string, a ...any) *sql.Row { return nil }
func (this *fakeTx) Commit() error                                                  { this.commits++; return nil }
func (this *fakeTx) Rollback() error                                                { this.rollbacks++; return nil }

type fakePool struct{ tx *fakeTx }

func (this *fakePool) PrepareContext(c context.Context, q string) (*sql.Stmt, error) {
	return nil, nil
}
func (this *fakePool) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return nil, nil
}
func (this *fakePool) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return nil, nil
}
func (this *fakePool) QueryRowContext(c context.Context, q string, a ...any) *sql.Row { return nil }
func (this *fakePool) BeginTx(c context.Context, o *sql.TxOptions) (gorm.ConnPool, error) {
	return this.tx, nil
}

func newClient(t *testing.T) (*gorm.DB, *fakeTx) {
	tx := &fakeTx{}

	client, err := gorm.Open(
		postgres.New(postgres.Config{Conn: &fakePool{tx: tx}}),
		&gorm.Config{DryRun: true},
	)

	assert.NoError(t, err)

	return client, tx
}

// --- BuildUpdateMap: the reason update daos exist ---

func TestBuildUpdateMapWritesZeroValues(t *testing.T) {
	name := ""
	verified := false
	count := 0

	assert.Equal(t,
		map[string]any{"Name": "", "VerifyEmail": false},
		repo.BuildUpdateMap(udto.UserUpdateDao{Name: &name, VerifyEmail: &verified}),
		"false and \"\" must reach the database, that is what the pointers are for",
	)

	assert.Equal(t,
		map[string]any{"VerificationCount": 0},
		repo.BuildUpdateMap(odto.OtpUpdateDao{VerificationCount: &count}),
	)
}

func TestBuildUpdateMapSkipsNilFields(t *testing.T) {
	name := "Luis"

	assert.Equal(t,
		map[string]any{"Name": "Luis"},
		repo.BuildUpdateMap(udto.UserUpdateDao{Name: &name}),
		"a nil pointer means the column is not touched",
	)

	assert.Empty(t,
		repo.BuildUpdateMap(udto.UserUpdateDao{}),
		"an empty dao must produce an empty map so Update can skip the query",
	)
}

// --- generated SQL ---

func TestUpdateWritesOnlyTheDaoColumnsPlusUpdatedAt(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	hash := "new-hash"

	statement := repository.Query().
		Where(entity.User{ID: 7}).
		Updates(repo.BuildUpdateMap(udto.UserUpdateDao{PasswordHash: &hash})).
		Statement

	// NOTE: `updated_at` only shows up because the dao goes through a map.
	// Passing the dao struct straight to Updates would silently skip it.
	assert.Equal(t,
		`UPDATE "users" SET "password_hash"=$1,"updated_at"=$2 WHERE "users"."id" = $3`,
		statement.SQL.String(),
	)
}

func TestListQueryPaginatesAndCountQueryDoesNot(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	option := repo.Option{With: []string{"Profile"}, Skip: 20, Limit: 10}

	var users []entity.User
	listing := repository.ListQuery(option).Where(entity.User{Name: "Luis"}).Find(&users).Statement
	assert.Contains(t, listing.SQL.String(), "LIMIT")
	assert.Contains(t, listing.SQL.String(), "OFFSET")

	var count int64
	counting := repository.Query(option).Where(entity.User{Name: "Luis"}).Count(&count).Statement
	assert.Equal(t,
		`SELECT count(*) FROM "users" WHERE "users"."name" = $1`,
		counting.SQL.String(),
		"the total must be the real total, never the page size",
	)
}

// captureQuery records the SQL of the next SELECT built on the client. DryRun
// still builds the statement and runs the callbacks, it just never executes.
func captureQuery(t *testing.T, client *gorm.DB, run func()) string {
	captured := ""

	err := client.Callback().Query().After("gorm:query").Register("test:capture", func(db *gorm.DB) {
		captured = db.Statement.SQL.String()
	})

	assert.NoError(t, err)
	defer client.Callback().Query().Remove("test:capture")

	run()

	return captured
}

func TestPageSizeDefaultsToTen(t *testing.T) {
	assert.Equal(t, repo.DefaultLimit, repo.Option{}.Size())
	assert.Equal(t, repo.DefaultLimit, repo.Option{Skip: 40}.Size())
	assert.Equal(t, 25, repo.Option{Limit: 25}.Size())
	assert.Equal(t, -1, repo.Option{Limit: -1}.Size(), "a negative limit is the opt out")
}

func TestPaginateIgnoresNonPositiveInput(t *testing.T) {
	// a request sending limit=-1 must not be able to disable the limit
	assert.Equal(t, repo.DefaultLimit, repo.Option{}.Paginate(0, -1).Size())
	assert.Equal(t, 0, repo.Option{}.Paginate(-5, 0).Skip)
	assert.Equal(t, repo.Option{Skip: 20, Limit: 5}, repo.Option{}.Paginate(20, 5))

	base := repo.Option{With: []string{"Profile"}}
	assert.Equal(t, []string{"Profile"}, base.Paginate(10, 10).With, "Paginate keeps the rest of the option")
	assert.Equal(t, 0, base.Skip, "Paginate returns a copy, it does not mutate")
}

func TestListingWithoutAPageUsesTheDefaultLimit(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	var users []entity.User

	statement := repository.ListQuery().Where(entity.User{Name: "Luis"}).Find(&users).Statement
	assert.Contains(t, statement.SQL.String(), "LIMIT")
	assert.Contains(t, statement.Vars, repo.DefaultLimit,
		"an unfiltered listing must never pull the whole table")

	statement = repository.ListQuery(repo.Option{Skip: 20, Limit: 5}).Find(&users).Statement
	assert.Contains(t, statement.Vars, 5)
	assert.Contains(t, statement.Vars, 20)
}

func TestNegativeLimitDropsTheLimitClause(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	var users []entity.User

	statement := repository.ListQuery(repo.Option{Limit: -1}).Find(&users).Statement

	assert.NotContains(t, statement.SQL.String(), "LIMIT")
}

func TestDefaultLimitDoesNotLeakIntoCountOrFindOne(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	var count int64
	counting := repository.Query().Where(entity.User{Name: "Luis"}).Count(&count).Statement
	assert.NotContains(t, counting.SQL.String(), "LIMIT",
		"the count must see every matching row")

	// FindOne goes through ListQuery, so the default limit is applied and then
	// overridden by First. The row cap must end up at 1, not 10.
	sql := captureQuery(t, client, func() {
		_, _ = repository.FindOne(entity.User{Email: "a@b.com"})
	})
	assert.Contains(t, sql, "LIMIT")

	statement := repository.ListQuery().Where(entity.User{Email: "a@b.com"}).
		First(&entity.User{}).Statement
	assert.Contains(t, statement.Vars, 1)
	assert.NotContains(t, statement.Vars, repo.DefaultLimit)
}

func TestFindLastOrdersByCreatedAtDescending(t *testing.T) {
	client, _ := newClient(t)
	repository := orep.NewOtpRepository(client)

	sql := captureQuery(t, client, func() {
		_, _ = repository.FindLast(entity.Otp{Contact: "a@b.com", AppId: "app-1"})
	})

	// The ordering belongs to the repository: a caller cannot forget it.
	assert.Contains(t, sql, `ORDER BY "created_at" DESC`)
	assert.Contains(t, sql, `"otps"."contact" = $`)
	assert.Contains(t, sql, "LIMIT")
}

func TestSortIsQuotedAndCannotInjectSql(t *testing.T) {
	client, _ := newClient(t)
	repository := urep.NewUserRepository(client)

	option := repo.Option{
		Sort: []repo.Sort{{Column: "created_at; DROP TABLE users", Direction: repo.Desc}},
	}

	var users []entity.User
	statement := repository.ListQuery(option).Find(&users).Statement

	assert.Contains(t, statement.SQL.String(), `"created_at; DROP TABLE users"`)
	assert.NotContains(t, statement.SQL.String(), "DROP TABLE users DESC")
}

// --- transaction routing ---

func TestOptionRoutesToTheTransaction(t *testing.T) {
	client, fake := newClient(t)
	manager := repo.NewTransactionManager(client)
	repository := urep.NewUserRepository(client)

	tx, err := manager.Tx()
	assert.NoError(t, err)

	assert.Equal(t, gorm.ConnPool(fake), repository.Query(repo.Option{Tx: tx}).Statement.ConnPool,
		"a call carrying the option must run on the transaction")

	assert.NotEqual(t, gorm.ConnPool(fake), repository.Query().Statement.ConnPool,
		"a call without the option must run on the pool")

	assert.NotEqual(t, gorm.ConnPool(fake), repository.Query(repo.Option{Tx: nil}).Statement.ConnPool,
		"a nil transaction must behave like no transaction")
}

func TestDeferredRollbackIsANoOpAfterCommit(t *testing.T) {
	client, fake := newClient(t)
	manager := repo.NewTransactionManager(client)

	tx, err := manager.Tx()
	assert.NoError(t, err)

	assert.NoError(t, tx.Commit())
	assert.NoError(t, tx.Rollback()) // the deferred call on the happy path

	assert.Equal(t, 1, fake.commits)
	assert.Equal(t, 0, fake.rollbacks)
}

func TestRollbackRunsOnAnEarlyReturn(t *testing.T) {
	client, fake := newClient(t)
	manager := repo.NewTransactionManager(client)

	func() {
		tx, err := manager.Tx()
		assert.NoError(t, err)

		defer tx.Rollback()
		// the service bails out before committing
	}()

	assert.Equal(t, 0, fake.commits)
	assert.Equal(t, 1, fake.rollbacks)
}

// --- update daos must stay in sync with their entity ---

func TestUpdateDaoKeysResolveToRealColumns(t *testing.T) {
	client, _ := newClient(t)

	name, phone := "Luis", ""
	verified, twoFactor := false, true
	organizationId, hash := "org-1", "hash"
	metadata := json.RawMessage(`{}`)

	dao := udto.UserUpdateDao{
		CurrentOrganizationId: &organizationId,
		Name:                  &name,
		Email:                 &name,
		Phone:                 &phone,
		VerifyEmail:           &verified,
		TwoFactorEnabled:      &twoFactor,
		PasswordHash:          &hash,
		Metadata:              &metadata,
	}

	statement := client.Model(&entity.User{}).Where("id = ?", 1).
		Updates(repo.BuildUpdateMap(dao)).Statement

	// An unknown key would be emitted as a raw column name and blow up in
	// postgres; every column below proves the dao field name resolved.
	for _, column := range []string{
		"current_organization_id", "name", "email", "phone",
		"verify_email", "two_factor_enabled", "password_hash", "metadata",
	} {
		assert.Contains(t, statement.SQL.String(), `"`+column+`"=`)
	}
}
