# Repository Pattern

> **Status: applied. Every repository follows this document.**
>
> `user`, `otp`, `app`, `session`, `user_pool` and `profile` are all migrated.
> This is the only valid way to write a repository — there is no legacy path
> left to fall back on.

## Why

The first generation of repositories accumulated four problems that this
pattern exists to fix:

1. **Query logic leaked into services.** `FindManyWhere(where any, args []any, ...)`
   let the caller build the SQL condition, so the same query was reinvented in
   several services and the repository stopped being the boundary it was
   supposed to be.
2. **Listing and counting were fused.** `FindManyWhereAndCount` returned
   `([]T, int64, error)`, forcing every caller to pay for a `COUNT` even when it
   only wanted the page, and making the two concerns impossible to reuse apart.
3. **No transaction support at all.** Every method used the pool client
   directly, so a unit of work spanning two tables could not be rolled back.
4. **`Update` was unsafe.** `Save(entity)` writes every column, and
   `Updates(entity)` silently drops `false`, `0` and `""` because gorm treats
   them as "unset".

## Building blocks

Everything lives in one place and is shared by every repository.

### `Option`

The single, optional, last argument of every repository action. It is flat on
purpose: nesting a `Filter` inside a `Page` inside the option turned every call
site into four lines of struct literal.

```go
type Option struct {
	Tx   *Tx
	With []string // relations to preload

	Skip  int
	Limit int
	Sort  []Sort
}
```

It is deliberately **not** an escape hatch. There is no `Where`, no `Conditions`
and no `args []any` field, because that is exactly how query logic escaped the
repository in the previous generation.

Ordering is applied through `clause.OrderByColumn`, which quotes the identifier,
so a column name arriving from a query string can never inject SQL.

Pagination always applies. `Option{}` means skip nothing and take
`repo.DefaultLimit` (10), so a listing that forgets to paginate reads one page
instead of the whole table. `Option.Size()` returns the size that will be used.
A negative `Limit` is the explicit opt out and drops the LIMIT clause — use it
only for internal jobs that genuinely need every row.

`Option.Paginate(skip, limit)` returns a copy with the page set, ignoring
anything not positive. Use it for values coming from a request: it is what stops
`?limit=-1` from disabling the limit. Set `Limit` on the struct directly only
for trusted internal callers.

### `BaseRepository[T, U]`

Implements the generic contract once, for every entity. `T` is the entity, `U`
is its update dao.

| Method                              | Returns             | Notes                                              |
| ----------------------------------- | ------------------- | -------------------------------------------------- |
| `Find(where T, opt...)`             | `([]T, error)`      | preloads + ordering + pagination                    |
| `FindCount(where T, opt...)`        | `(int64, error)`    | ignores pagination and preloads                     |
| `FindOne(where T, opt...)`          | `(*T, error)`       | `nil, nil` when nothing matches                     |
| `Create(value T, opt...)`           | `(*T, error)`       | returns the row with generated columns filled       |
| `Update(where T, dao U, opt...)`    | `(int64, error)`    | rows affected                                       |
| `Delete(where T, opt...)`           | `(int64, error)`    | rows affected                                       |
| `Query(opt...)`                     | `*gorm.DB`          | table + transaction, for dedicated queries          |
| `ListQuery(opt...)`                 | `*gorm.DB`          | `Query` + `Option.Apply` (preloads, order, page)    |

A concrete repository embeds it and adds only what is specific to the entity.

`repo.NoUpdate` is the update dao of a read only entity, such as the
`app_role_profiles` reference table: `BaseRepository[entity.X, repo.NoUpdate]`.
Its `Update` always resolves to an empty map and touches nothing, and the
repository interface simply does not declare `Update`.

Two package level helpers exist for dedicated queries:

- `FirstOrNil[T](query)` — runs `First` and folds "record not found" into
  `(nil, nil)`, so a dedicated single row query matches the `FindOne` contract
  without re-handling `ErrRecordNotFound`.
- `IsNotFound(err)` — the gorm check, so nothing outside `shared/repository`
  imports gorm for it.

### `Tx` and `ITransactionManager`

```go
type ITransactionManager interface {
	Tx() (*Tx, error)
}
```

`Tx` exposes only `Commit()` and `Rollback()`. A service never receives, imports
or names `*gorm.DB` — the raw client stays inside the repository layer and the
transaction manager.

Every repository method routes through the same three lines:

```go
func (this Option) Session(client *gorm.DB) *gorm.DB {
	if session := this.Tx.session(); session != nil {
		return session
	}
	return client
}
```

If the option carries a transaction, the statement runs on it. If not, it runs
on the pool. That is the whole mechanism.

### Update dao

A per entity struct listing the columns that are allowed to change, every field
a pointer:

```go
type UserUpdateDao struct {
	Name        *string
	Email       *string
	VerifyEmail *bool
	// ...
}
```

`nil` means "do not touch". A filled pointer is written as is, including
`false`, `0` and `""`. `BuildUpdateMap` converts it to the `map[string]any` that
gorm receives (see the edge case below for why a map and not the struct).

## How to add a new repository

1. **Update dao** — `app/modules/{scope}/{module}/models/{module}.dao.go`
   (see [models-layer.md](models-layer.md)):

   ```go
   type SessionUpdateDao struct {
       Invalidated *bool
       LastUsedAt  *time.Time
   }
   ```

   Field names must match the entity field names. Only list columns that are
   legitimately mutable — this struct is the allow list.

2. **Interface** — `repository/interface.go`. Declare the generic methods you
   actually use plus every dedicated query. The interface is the gate: methods
   promoted from `BaseRepository` that are not declared here are unreachable
   from services.

   ```go
   type ISessionRepository interface {
       FindOne(where entity.Session, options ...repo.Option) (*entity.Session, error)
       Create(session entity.Session, options ...repo.Option) (*entity.Session, error)
       Update(where entity.Session, dao dto.SessionUpdateDao, options ...repo.Option) (int64, error)
   }
   ```

3. **Implementation** — `repository/{module}.repository.go`:

   ```go
   type SessionRepository struct {
       repo.BaseRepository[entity.Session, dto.SessionUpdateDao]
   }

   var _ ISessionRepository = &SessionRepository{}

   func NewSessionRepository(client *gorm.DB) *SessionRepository {
       return &SessionRepository{
           BaseRepository: repo.NewBaseRepository[entity.Session, dto.SessionUpdateDao](client),
       }
   }
   ```

4. **Dedicated queries**, when the typed `where` cannot express the condition.
   Keep the naming pair and always accept the option:

   ```go
   func (this *UserRepository) FindFromAppId(appId string, options ...repo.Option) ([]entity.User, error) {
       result := []entity.User{}
       err := this.ListQuery(options...).
           Joins(userFromAppJoin).
           Where("apps.id = ?", appId).
           Find(&result).Error
       return result, err
   }

   func (this *UserRepository) FindFromAppIdCount(appId string, options ...repo.Option) (int64, error) {
       var count int64
       err := this.Query(options...).
           Joins(userFromAppJoin).
           Where("apps.id = ?", appId).
           Count(&count).Error
       return count, err
   }
   ```

   Use `ListQuery` for the listing (it applies preloads, ordering and
   pagination) and `Query` for the count (it does not).

   A dedicated query returning a single row uses `FirstOrNil`, and bakes the
   ordering in so a caller cannot forget it:

   ```go
   func (this *OtpRepository) FindLast(where entity.Otp, options ...repo.Option) (*entity.Otp, error) {
       query := this.ListQuery(options...).
           Where(where).
           Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true})

       return repo.FirstOrNil[entity.Otp](query)
   }
   ```

   "The most recent one" is an ordering decision, which is query logic, so it
   lives here and not in the option the service builds.

## How to use it from a service

The rules on the service side — wrapping errors, checking for `nil`, owning the
transaction, never importing gorm — are in
[service-layer.md](service-layer.md). Where the update dao file lives and how it
is named is in [models-layer.md](models-layer.md).

### Read

```go
user, err := this.userRepository.FindOne(entity.User{Email: email})

if err != nil {
    return nil, e.ThrowInternalServerError("Failed to find user")
}

if user == nil {
    return nil, e.ThrowNotAllowed("User not found")
}
```

### Read with relations and pagination

```go
option := repo.Option{
    With: []string{"Profile"},
    Sort: []repo.Sort{{Column: "created_at", Direction: repo.Desc}},
}

if query != nil {
    option = option.Paginate(query.Skip, query.Limit)
}

users, err := this.userRepository.FindFromAppId(appId, option)
if err != nil {
    return nil, err
}

total, err := this.userRepository.FindFromAppIdCount(appId, option)
if err != nil {
    return nil, err
}
```

Report `option.Skip` back to the client rather than the raw `query.Skip`: it is
the value that was actually applied after sanitising.

Listing and counting are two calls, combined here in the service. The option is
reused as is: `FindCount` ignores the pagination on purpose, so the total is the
total and not the page size.

The service does not default the page size itself — `Paginate` ignores
non-positive values and the repository applies `repo.DefaultLimit`, so the
default lives in exactly one place.

### Single mutation

No transaction. gorm already wraps an isolated `Create`/`Update`/`Delete` in its
own transaction.

```go
verified := true

affected, err := this.userRepository.Update(
    entity.User{ID: userId},
    models.UserUpdateDao{VerifyEmail: &verified},
)
```

### Several mutations in one unit of work

```go
tx, err := this.txManager.Tx()

if err != nil {
    return nil, e.ThrowInternalServerError("Failed to open transaction")
}

defer tx.Rollback()

option := repo.Option{Tx: tx}

user, err := this.userRepository.Create(entity.User{...}, option)
if err != nil {
    return nil, err
}

if _, err := this.sessionRepository.Create(entity.Session{UserId: user.ID}, option); err != nil {
    return nil, err
}

if err := tx.Commit(); err != nil {
    return nil, e.ThrowInternalServerError("Failed to commit transaction")
}
```

`defer tx.Rollback()` goes immediately after `Tx()`, always. It is a no op after
a successful commit, and it is what prevents an early `return` from leaking the
connection.

When a call inside the unit of work also needs relations, fill the same literal:

```go
created, err := this.userRepository.FindOne(
    entity.User{ID: user.ID},
    repo.Option{Tx: tx, With: []string{"Profile"}},
)
```

## Rules

- Query logic lives in the repository. A service never builds a condition, a
  join or an order clause.
- Listing and counting are separate methods, combined in the service.
- Dedicated queries keep the `X` / `XCount` pair and always accept
  `options ...Option`, so they work inside a transaction and with
  preloads like any other method.
- A transaction is only for a unit of work with more than one mutation.
- Update goes through the update dao. `Save(entity)` and `Updates(entity)` are
  not used anywhere. `Save` in particular rewrites every column of the row, so
  two concurrent writers clobber each other.
- A repository method that exists only to flip one column (the old
  `Invalidate(otpId)`) is not needed: the update dao already writes `false`,
  `0` and `""`, so the generic `Update` covers it.
- The repository interface is the contract. Do not declare a `BaseRepository`
  method the module does not need.

## Edge cases

### gorm ignores zero values in a struct `where`

`Find(entity.User{VerifyEmail: false})` does **not** filter by
`verify_email = false` — gorm skips zero valued fields when building conditions
from a struct, so that call means "all users". The same applies to `0` and `""`.

This is a deliberate limitation, not an oversight: the fix is a dedicated
repository method, not a raw condition passed by the service.

```go
func (this *UserRepository) FindUnverified(usersPoolId string, options ...repo.Option) ([]entity.User, error) {
	result := []entity.User{}
	err := this.ListQuery(options...).
		Where("users_pool_id = ? AND verify_email = ?", usersPoolId, false).
		Find(&result).Error
	return result, err
}
```

### `Updates(dao)` silently skips `updated_at`

Verified against gorm v1.31.1 (`callbacks/update.go`, `ConvertToAssignments`):

- **map branch** — auto fills every `AutoUpdateTime` column that is not already
  in the map.
- **struct branch** — walks the model columns and looks each one up in the
  *destination* schema. A dao without an `UpdatedAt` field means the column is
  never found, so it is never written.

Passing the dao struct straight to `Updates` therefore produces:

```
UPDATE "users" SET "name"=$1,"verify_email"=$2,"metadata"=$3 WHERE id = $4
```

while going through `BuildUpdateMap` produces:

```
UPDATE "users" SET "metadata"=$1,"name"=$2,"verify_email"=$3,"updated_at"=$4 WHERE id = $5
```

This is the entire reason `Update` converts the dao to a map. Do not "simplify"
it back to `Updates(dao)`.

### Update dao field names must match the entity

`BuildUpdateMap` emits the Go field name as the key and lets gorm resolve it to
the column through the model schema, so the naming strategy is never duplicated.
A field that does not exist on the entity is passed through as a raw column
name and fails at the database with an unknown column error — loud, not silent,
but still worth catching in review.

### Only the first option is read

`ResolveOption` takes `options[0]` and ignores the rest. Passing two options is
not a merge, it is a silently discarded second argument. The variadic form
exists only to make the option optional.

### `FindOne` returns `nil, nil`

Not found is not an error. Services check `if user == nil`, never
`errors.Is(err, gorm.ErrRecordNotFound)`, and no service imports gorm for this.
A non nil error always means a real failure.

### `Update` and `Delete` with an empty `where`

gorm refuses a global update or delete with `ErrMissingWhereClause`. This is a
safety net worth keeping, not something to work around with `AllowGlobalUpdate`.

### An update dao with every field nil

`BuildUpdateMap` returns an empty map and `Update` short circuits to `(0, nil)`
without touching the database. `RowsAffected == 0` therefore means either "no
row matched" or "nothing to update" — check the dao first if the distinction
matters.

### A dedicated query may opt out of pagination

`ProfileRepository.FindByAppRole` builds on `Query`, not `ListQuery`, because
the caller resolves one winning profile out of the whole set and a truncated
result would silently pick the wrong one. That is a legitimate choice for a
small reference table, but it must be a deliberate, commented one — the default
is `ListQuery`.

### Pagination is never optional

`Option.Apply` always emits a LIMIT. `Option{}` resolves to `DefaultLimit` (10),
which means `Find(where)` with no option returns at most 10 rows — not the
whole table. Anything that needs a different page size must say so, and
anything that genuinely needs every row must pass a negative `Limit`.

This applies to listings only. `FindCount` goes through `Query`, so the total is
never capped, and `FindOne` ends up at `LIMIT 1` because gorm's `First`
overrides the default.

### Preloads and counting

`FindCount` uses `Query`, not `ListQuery`, so it never applies preloads or
pagination. A count on a joined query can over count if the join can match more
than one row per entity; `FindFromAppIdCount` is safe because `apps.id` is a
primary key, but a new dedicated count over a one-to-many join needs
`Distinct`.

### Forgetting the option inside a transaction

A repository call inside a `Tx()` block that does not receive the option runs on
the pool: it is not rolled back, and it can block waiting on rows the open
transaction is holding. The compiler cannot catch this. Rule of thumb: inside a
transaction, every repository call carries the option.

## Layout

| What                                             | Where                                       |
| ------------------------------------------------ | ------------------------------------------- |
| `Option`, `Sort`, `DefaultLimit`                 | `shared/repository/option.go`               |
| `Tx`, `ITransactionManager`, `TransactionManager` | `shared/repository/transaction.go`          |
| `IUpdateDao`, `BuildUpdateMap`                   | `shared/repository/update.go`               |
| `BaseRepository`, `FirstOrNil`, `IsNotFound`     | `shared/repository/base.go`                 |
| `{Module}UpdateDao`                              | `app/modules/{scope}/{module}/models/{module}.dao.go` |
| `I{Module}Repository`, `{Module}Repository`      | `app/modules/{scope}/{module}/repository/`  |
| `NewTransactionManager` provider                 | `cmd/main.go`                               |
| Pattern tests                                    | `tests/shared/repository_test.go`           |
| Dedicated query tests                            | `tests/shared/dedicated_queries_test.go`    |
| DI graph validation                              | `cmd/main_test.go`                          |

`shared/interfaces` holds only pure interfaces (`IController`, `IGuard`).

The import alias for the shared package is `repo`, since a module's own
repository package is also called `repository`:

```go
import repo "auth_service/shared/repository"
```

## Tests

[tests/shared/repository_test.go](../tests/shared/repository_test.go) locks in
the behaviours this document depends on, against a fake conn pool in gorm
`DryRun` mode (no database needed): zero values reaching the database, nil
fields being skipped, `updated_at` being written, counting ignoring pagination,
`Sort` columns being quoted, `FindLast` ordering, transaction routing, and
`defer tx.Rollback()` being a no op after commit.

`dedicated_queries_test.go` covers the per module queries: the session
`invalidated = false` predicate, the app search scopes, and the profile lookup
staying unpaginated.

`cmd/main_test.go` runs `fx.ValidateApp` over the whole container, so a
constructor signature change that breaks the wiring fails in tests instead of
at boot.

Run them with `go test ./tests/shared/ ./cmd/`.

### The bug this pattern found

`AuthorizeService` used to look a session up with
`FindWhere(entity.Session{ID: id, Invalidated: false})`. gorm drops zero valued
fields from a struct condition, so `Invalidated: false` compiled to nothing and
the query was just `WHERE sessions.id = $1`. Neither caller checked
`session.Invalidated` afterwards, so an invalidated session still authorized as
long as its token matched. `SessionRepository.FindActive` now spells the
predicate out. This is the concrete reason the zero value edge case is the first
one in this section.
