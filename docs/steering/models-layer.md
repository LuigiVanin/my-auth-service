# Models Layer

`models/` is where a module declares the shapes that cross its boundaries. It
holds two different kinds of type, in two files, with different rules.

```
{module}/models/
├── {module}.dto.go   # what crosses the HTTP boundary
└── {module}.dao.go   # what crosses the database boundary
```

Neither file may import a repository, a service or a controller. Both are
imported with the alias `dto` (or a prefixed one such as `udto`, `odto` when a
file needs several).

## Where each shape lives

| Shape                              | Where                              | Why                                    |
| ---------------------------------- | ---------------------------------- | --------------------------------------- |
| Request body, query, response      | `{module}/models/{module}.dto.go`  | transport contract                      |
| Update dao                         | `{module}/models/{module}.dao.go`  | database write contract                 |
| Database table                     | `infra/entities/{entity}.entity.go`| GORM entity, shared by every module     |
| Payload shared by several modules  | `shared/models/`                   | `RequestInfo` (ip, user agent)          |
| Result shaped by a service, not by HTTP | `services/interface.go`       | `VerifyConsumableOtpResponse`           |
| Response body with no reuse        | unexported struct in the controller| exists only for the OpenAPI schema      |

Entities are **not** models. They live in `infra/entities/` and a module never
declares one in its own folder.

## DTOs

### Request bodies

Validation tags are the contract. `middleware.BodyValidator[T]()` runs them and
answers a 400 with a `fields` array before the handler is reached.

```go
type CreateAppPayload struct {
    Name       string   `json:"name" validate:"required"`
    LoginTypes []string `json:"login_types" validate:"required,dive,oneof=WITH_LOGIN WITH_OTP WITH_PASSWORD"`
    TokenType  string   `json:"token_type" validate:"required,oneof=JWT FAST_JWT SESSION_UUID"`

    TokenExpirationTime int64 `json:"token_expiration_time" validate:"required,numeric,gt=0"`

    UserPool CreateAppPayloadUserPool `json:"user_pool" validate:"required"`
}
```

- `json` names are `snake_case`, matching the API.
- Enumerations are enforced with `oneof`, not checked in the service.
- A conditional pair uses `required_without` (see `CreateAppPayloadUserPool`).
- `validate:"required"` on a `bool` means "must be true", not "must be present".
  It is used that way on `VerifyEmail` today; prefer `*bool` when the field is
  genuinely optional.

When the body shape depends on a query parameter, declare one DTO per variant
(`LoginPayloadWithPassoword`, `LoginPayloadWithOtp`) and let a validator in
`app/middlewares/validators/` pick between them.

### Query structs

Bound with `ctx.Bind().Query(&query)`, tagged with `query`:

```go
type GetAppsQuery struct {
    Skip  int    `query:"skip"`
    Limit int    `query:"limit"`
    Name  string `query:"name"`

    OwnerUserId int64 `query:"owner_user_id"`
}
```

Do not default `Skip` and `Limit` here or in the service — forward them to
`repo.Option.Paginate(skip, limit)`, which ignores non positive values and lets
the repository apply `repo.DefaultLimit`. That also stops `?limit=-1` from
disabling the page cap. See [repository-pattern.md](repository-pattern.md).

### Response payloads

A listing response follows the same shape everywhere:

```go
type GetAppsResponse struct {
    Total  int64        `json:"total"`   // rows matching the filter, ignoring the page
    Amount int          `json:"amount"`  // rows in this page
    Skip   int          `json:"skip"`    // page offset actually applied
    Limit  int          `json:"limit"`   // page size actually applied
    Data   []entity.App `json:"data"`
}
```

Fill `Skip` and `Limit` from `option.Skip` and `option.Size()`, not from the raw
query — those are the values that were actually applied after sanitising.

Returning an entity directly is fine when the entity's `json` tags already hide
what must not leak (`User.PasswordHash` is `json:"-"`). When they do not, blank
the fields before answering, as `AppService.FindAll` does with the keys.

## DAOs

A dao is the allow list of columns a repository is permitted to write. Every
field is a pointer, so `nil` means "do not touch" and a filled pointer writes
the value — including `false`, `0` and `""`.

```go
package models

// UserUpdateDao is the allow list of updatable columns of entity.User.
type UserUpdateDao struct {
    ProfileId        *string
    Name             *string
    Email            *string
    Phone            *string
    VerifyEmail      *bool
    TwoFactorEnabled *bool
    PasswordHash     *string
    Metadata         *json.RawMessage
}
```

Rules:

- **Field names must match the entity field names.** `BuildUpdateMap` emits the
  Go field name and lets gorm resolve it to the column, so a mismatch fails at
  the database with an unknown column error.
- **List only what is legitimately mutable.** The dao is a security boundary as
  much as a convenience: a column absent from it can never be written through
  `Update`. `SessionUpdateDao` has two fields, not fifteen.
- **No gorm tags.** They belong on the entity. A dao copying `uniqueIndex` from
  the entity is noise that will drift.
- **A read only entity uses `repo.NoUpdate`** instead of an empty dao — see
  `ProfileRepository`.
- A dao may implement `ToUpdateMap() map[string]any` to normalize values
  (lowercase an email); `BuildUpdateMap` honours the override.

The full rationale for pointers, and the gorm behaviour that makes them
necessary, is in [repository-pattern.md](repository-pattern.md).

## Entities

`infra/entities/{name}.entity.go`, one per table, with an explicit `TableName()`:

```go
type User struct {
    ID    uint   `gorm:"primaryKey;autoIncrement" json:"id"`
    Email string `gorm:"not null;uniqueIndex:users_email_users_pool_unique,priority:1" json:"email"`
    PasswordHash string `gorm:"not null" json:"-"`

    UsersPool *UsersPool `gorm:"foreignKey:UsersPoolId" json:"-"`
    Profile   *Profile   `gorm:"foreignKey:ProfileId" json:"profile,omitempty"`
}

func (User) TableName() string { return "users" }
```

- Entities carry both `gorm` and `json` tags: they are serialized straight into
  responses in several places.
- Anything secret is `json:"-"` — `PasswordHash`, `Session.Token`,
  `Session.RefreshToken`, `Otp.Code`.
- Relations are preloaded through `repo.Option.With`, by field name
  (`"Profile"`, `"User.Profile"`), never by joining in a service.
- Schema changes are applied with GORM AutoMigrate (`make migrate up`).

## Naming

| Type              | Suffix / shape                | Example                     |
| ----------------- | ----------------------------- | --------------------------- |
| Request body      | `...Payload`                  | `CreateAppPayload`          |
| Query struct      | `...Query`                    | `GetAppsQuery`              |
| Response body     | `...Response`                 | `GetUsersAppResponse`       |
| Update dao        | `{Entity}UpdateDao`           | `UserUpdateDao`             |
| Repository search | `{Entity}Search`              | `AppSearch`                 |
| Entity            | singular, no suffix           | `User`, `App`, `UsersPool`  |

Note `Passoword` is misspelled in `LoginPayloadWithPassoword` and
`RegisterPayloadWithPassoword`. Do not propagate the typo into new names; the
existing ones are left alone because they are referenced across modules and
tests.
