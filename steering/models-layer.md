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

### Update bodies

**Every field of an update DTO is a pointer.** Go has no null for a value type, so
`*T` is the only way to tell "the caller did not send this field" from "the caller
sent the zero value". Without it `{"private": false}` and a body with no `private`
at all are the same struct, and a `PUT` either cannot write `false`, `0` and `""`
or overwrites every column the caller did not mention.

```go
type UpdateApp struct {
    Name       *string   `json:"name" validate:"omitnil,min=1"`
    LoginTypes *[]string `json:"login_types" validate:"omitnil,min=1,dive,oneof=WITH_LOGIN WITH_OTP WITH_PASSWORD"`

    TokenExpirationTime *int64 `json:"token_expiration_time" validate:"omitnil,gt=0"`

    Private *bool `json:"private"`

    Metadata *json.RawMessage `json:"metadata"`
}
```

- `omitnil`, not `omitempty`. Both work — `hasValue` special cases a pointer, so
  `omitempty,gt=0` does still run on a `*int64` pointing at `0` — but `omitnil`
  says the intent and saves the next reader that verification.
- A slice is already a reference type, but it is still declared `*[]string`, so
  the rule reads the same on every field.
- `min=1` next to a `dive`: an explicit `[]` passes `hasValue` and then `dive`
  validates nothing, which is how an app with no login method would get written.
- The dao mirrors the DTO, and it is the narrower of the two: a column absent
  from the dao can never be written whatever the payload says.

The service maps the payload onto the dao field by field and hands it to one
`Update`. Before answering `404` on `RowsAffected == 0` it has to ask
`repo.HasChanges(dao)`, because an all-nil payload resolves to an empty map and
`Update` short circuits to `(0, nil)` — see
[repository-pattern.md](repository-pattern.md).

### Open JSON columns are merged, never replaced

A `metadata` column is a document the caller owns keys inside of, not a value it
replaces. An update therefore **merges** the patch into what is stored, with
[RFC 7386](https://www.rfc-editor.org/rfc/rfc7386) merge patch semantics:

| Patch | Effect |
| --- | --- |
| `{"a": 1}` on `{"b": 2}` | `{"a": 1, "b": 2}` — keys not named are kept |
| `{"a": "new"}` on `{"a": "old"}` | `{"a": "new"}` — the request wins |
| `{"a": null}` | the key is **removed**; this is the only way to delete one |
| `{"outer": {"x": 1}}` on `{"outer": {"y": 2}}` | merged one level down, recursively |
| `{"list": [3]}` on `{"list": [1, 2]}` | replaced — an array is a value, not a document |

`utils.MergeJsonPatch(stored, patch)` is the single implementation.

**It is called from the service, never from the repository.** The repository
writes the columns it is given; deciding that two documents become one is
processing of what the caller sent, which is the service's job. A repository that
merged would also have to read the row first, turning every update into a
read-modify-write it has no business owning.

Two things the mechanism cannot do, both deliberate:

- **Clearing the whole object is not expressible.** `{"metadata": null}` leaves a
  `*json.RawMessage` field `nil`, indistinguishable from an absent one:
  `encoding/json` zeroes the pointer before it ever reaches `RawMessage`'s
  `UnmarshalJSON`, and fiber uses `encoding/json`. So `null` on the column means
  "do not touch", and deletion is per key.
- **A patch that is not an object is refused**, not applied. RFC 7386 would
  replace the document with the scalar; every jsonb column here is documented as
  an object, so `MergeJsonPatch` errors and the service answers `400`.

### A column the service owns

Data the service maintains itself does **not** go into `metadata`. It gets a column
of its own, absent from the update dao, written only by a named repository method —
`users.tracking` and `users_pool.tracking` are the reference:

```go
// infra/entities/user.entity.go
Tracking json.RawMessage `gorm:"type:jsonb;default:'{}';not null" json:"-"`
```

```go
// The only writer. `tracking` is not a dao field, so no payload reaches it.
WriteTracking(id uint, document json.RawMessage, options ...repo.Option) (int64, error)
```

The protection is the dao rule stated above, not a new one: leave the column out of
the dao and no request body can write it, whatever it sends. This replaced an
earlier attempt at a "reserved key" refused inside `metadata`, which needed a check
on every write path to hold — the column needs none.

Two details that come with it:

- **Serialization is a decision, not a default.** A column embedded in a widely
  reused entity leaks everywhere that entity does. `users.tracking` is `json:"-"`
  and comes back only from `dto.GetUserResponse`, which shadows the embedded field;
  `users_pool.tracking` serializes and the listing blanks it row by row.
- **`omitempty` does not hide a jsonb column.** It omits `json.RawMessage` only at
  length zero, and `default:'{}'` is two bytes — so it hides only a value the code
  zeroed on purpose.

Where the write is a counter or a document changed in place, the concurrency rules
are in [repository-pattern.md](repository-pattern.md) and the worked case is
`docs/features/2026-09-09-tracking/spec.md`.

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
| Request body      | no suffix, named by the action | `CreateProfile`, `SwitchOrganization` |
| Query struct      | `...Query`                    | `GetAppsQuery`              |
| Response body     | `...Response`                 | `GetUsersAppResponse`       |
| Update dao        | `{Entity}UpdateDao`           | `UserUpdateDao`             |
| Repository search | `{Entity}Search`              | `AppSearch`                 |
| Entity            | singular, no suffix           | `User`, `App`, `UsersPool`  |

The `dto` import alias already says these are transport types, so a `Payload` or
`Dto` suffix only adds noise: `dto.CreateProfile` reads better than
`dto.CreateProfilePayload`. `CreateUserPool` was already written this way.

**Most request bodies still carry the old `...Payload` suffix** — `CreateAppPayload`,
`CreateOrganizationPayload`, `SwitchOrganizationPayload`, `ResetPasswordPayload`,
`VerifyConsumableOtpPayload`, and the four `Login`/`Register` variants. Renaming them
is roughly ninety occurrences across twenty five files, touching controllers,
validators, services, OpenAPI registrations and tests. It has not been done. New
request bodies follow the rule above; the old names are left alone until someone
decides to migrate them in one pass.

Note that `...Payload` is still the right suffix for a structure that is *not* an HTTP
request body — `OtpMetadataPayload` and `OtpStoredMetadataPayload` describe what is
stored inside an OTP, and the rule above does not reach them.

Note `Passoword` is misspelled in `LoginPayloadWithPassoword` and
`RegisterPayloadWithPassoword`. Do not propagate the typo into new names; the
existing ones are left alone because they are referenced across modules and
tests.
