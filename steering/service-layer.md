# Service Layer

A service holds the business rules of a module. It is the only layer that
decides *what should happen*: the controller decides how the request is read,
the repository decides how the data is fetched.

## When to create one

- A **core module** gets a service when its rules are used by more than one
  caller, or when they are more than a passthrough to the repository.
  `UserService.IsAlreadyCreated` exists because three modules need the same
  answer; `ProfileService.GetProfileByAppRole` exists because picking the
  winning profile out of the role rows is a rule, not a query.
- An **api module** always gets a service. The controller stays a thin adapter.
- A **utils module** is a service with no repository behind it (`cipher`,
  `hash`, `jwt`).

If a controller would call the repository directly, that is the signal a service
is missing. A controller must never hold a repository — `UserPoolController`
used to, and the fix was to extract `UserPoolService`.

## Anatomy

`services/interface.go` declares the contract, and any response type that is
shaped by the service rather than by the transport:

```go
package services

type IUserService interface {
    IsAlreadyCreated(email string, app *entity.App) (bool, error)
    FindUserInPool(email string, usersPoolId string) (*entity.User, error)
    Update(where entity.User, data dto.UserUpdateDao) (int64, error)
    FindAllUsersFromApp(...) (*dto.GetUsersAppResponse, error)
}
```

`services/{module}.service.go` holds the implementation:

```go
type UserService struct {
    userRepository ur.IUserRepository
    appRepository  ar.IAppRepository
}

var _ IUserService = &UserService{}

func NewUserService(userRepository ur.IUserRepository, appRepository ar.IAppRepository) *UserService {
    return &UserService{
        userRepository: userRepository,
        appRepository:  appRepository,
    }
}

func (this *UserService) IsAlreadyCreated(email string, app *entity.App) (bool, error) {
    // ...
}
```

- Constructor is `NewXxxService`, takes interfaces, returns the concrete type.
  FX binds it with `fx.As`.
- `var _ IXxxService = &XxxService{}` is mandatory — it is what makes an
  interface drift a compile error.
- The receiver is `this`, project wide.
- Dependencies are stored as interfaces, never as concrete types.

## Rules

**Return `*AppError`, never a raw error.** Everything a service returns to a
controller is either `nil` or a typed error from `app/errors`. Wrap what comes
from below rather than forwarding it:

```go
users, err := this.userRepository.FindFromAppId(targetAppId, option)

if err != nil {
    return nil, e.ThrowInternalServerError("Failed to fetch users")
}
```

See [error-handling.md](error-handling.md) for choosing the code.

**Never import gorm.** Not for the client, not for `ErrRecordNotFound`, not for
a transaction handle. `FindOne` reports "not found" as `nil, nil`, and
transactions come from `repo.ITransactionManager` which hands out an opaque
`*repo.Tx`. A `gorm` import in a service is a bug.

**Never build query logic.** No conditions, no joins, no order clauses, no raw
SQL strings passed down. If the typed `where` cannot express it, the repository
gets a dedicated method. This is the rule the old `FindManyWhereAndCount(query
any, args []any, ...)` broke, and it is how `AppService.FindAll` ended up
listing every application in the database to any authenticated user.

**Check for `nil` after a `FindOne`.** A `nil` error and a `nil` pointer means
the row does not exist. Deciding whether that is a 404, a `false`, or a normal
branch is a business decision and therefore belongs here:

```go
user, err := this.userRepository.FindOne(entity.User{Email: email})

if err != nil {
    return nil, e.ThrowInternalServerError("Failed to find user")
}

if user == nil {
    return nil, e.ThrowNotFound("User not found in User Pool")
}
```

**Own the transaction.** A unit of work with more than one mutation opens the
transaction in the service and threads the option through every call inside it.
Read only flows and single mutations do not need one. See
[repository-pattern.md](repository-pattern.md).

**Combine listing and counting here.** Repositories expose `Find` and
`FindCount` separately; assembling the page response is the service's job.

**Authorization decisions belong here, not in the controller.**
`FindAllUsersFromApp` and `FindAll` both resolve what the caller's profile is
allowed to see. The `PermissionsGuard` covers route level access; row level
visibility is a business rule.

## Depending on other layers

| Dependency                     | Allowed | Note                                              |
| ------------------------------ | ------- | -------------------------------------------------- |
| Its own module's repository    | yes     | the normal case                                     |
| Another module's **service**   | yes     | `RegisterService` → `IOtpService`, `ISessionService` |
| Another module's **repository**| no      | go through that module's service                    |
| A controller                   | no      | never                                               |
| `*gorm.DB`                     | no      | never                                               |
| `*zap.Logger`, `*config.Config`| yes     | injected like any other dependency                  |

A service that needs a repository from another module is the signal that the
other module is missing a service method.

## Logging

Inject `*zap.Logger` and log with structured fields. Log what the error handler
cannot see — the internal cause — and let the handler log the request itself:

```go
this.logger.Error("Failed to look up the last OTP", zap.Error(err))
return nil, e.ThrowInternalServerError("Failed to check OTP rate limit")
```

Do not log and return the same string; the handler already logs every error with
method, path, code and status.

## Gotchas seen in this codebase

- **Do not swallow repository errors.** `OtpService.GenerateConsumable` used to
  guard its rate limit with `if err == nil && lastOtp != nil`, which meant a
  database failure silently disabled the rate limit. A real error must surface.
- **Do not mutate an entity and save it back.** Updates go through the update
  dao so only the changed column is written. `Save(entity)` rewrites every
  column, so two concurrent writers clobber each other.
- **Keep the log after the error check.** `RegisterService` logged "User created
  Successfully!" before checking whether the create had failed.
