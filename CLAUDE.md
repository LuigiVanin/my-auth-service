# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

It is an index, not a manual. Each layer has a dedicated document under
`docs/steering/` — read the relevant one **before** writing code in that layer.

## Project Overview

A robust authentication and authorization service built with Go 1.25.3, handling
user management, multi-tenancy, and fine-grained access control using clean
architecture principles.

**Tech Stack:**
- Go 1.25.3
- Fiber v3 (web framework)
- PostgreSQL with GORM
- Uber FX (dependency injection)
- Zap (structured logging)
- JWT (golang-jwt/jwt)
- Resend (email service)
- Air (hot reload in development)

## Steering Documents

Detailed conventions live in `docs/steering/`. **Read the document for a layer before
adding to or changing that layer** — each one carries decisions, rules and edge
cases that are not obvious from the code.

| Document | Covers | Read it when |
| -------- | ------ | ------------ |
| [module-structure.md](docs/steering/module-structure.md) | Module anatomy, the api/core/utils split, FX wiring, registration order | Creating a module, wiring a provider, changing a constructor |
| [controller-layer.md](docs/steering/controller-layer.md) | HTTP handlers, guard chains, route registration, OpenAPI docs | Adding or changing an endpoint |
| [guards-and-middlewares.md](docs/steering/guards-and-middlewares.md) | The request chain, the five guards, permission documents, validators | Adding a guard or middleware, changing who may call a route |
| [service-layer.md](docs/steering/service-layer.md) | Business logic, dependency rules, transactions, error wrapping | Adding or changing business rules |
| [repository-pattern.md](docs/steering/repository-pattern.md) | `repo.Option`, `BaseRepository`, update daos, transactions, dedicated queries | Touching anything that reads or writes the database |
| [models-layer.md](docs/steering/models-layer.md) | DTOs, update daos, entities, validation tags, naming | Declaring a payload, a query struct, a dao or an entity |
| [profiles.md](docs/steering/modules/profiles.md) | Permission documents, the pool → organization → participant hierarchy, `permissions.Resolve`, clamp vs refuse | Touching a permission document, a profile, or anything that reads permissions |
| [error-handling.md](docs/steering/error-handling.md) | `AppError`, error codes, problem details, when to add a new error | Throwing, reusing or creating an error |
| [plugins.md](docs/steering/plugins.md) | Cross cutting `fx.Option` features, invoke ordering | Adding behaviour that spans every module |

### Which layer am I in?

- **Reading or writing the database** → repository. Query logic never leaves it.
- **Deciding what should happen** → service. Business rules, authorization of
  rows, transactions, orchestration across modules.
- **Writing inside a transaction** → the repository, even another module's.
  Reaching for the service there either drops the write out of the transaction or
  drags `repo.Option` across a service boundary. See
  [service-layer.md](docs/steering/service-layer.md).
- **Reading a request, writing a response** → controller. Thin adapter, one
  service call, no logic.
- **Declaring a shape that crosses a boundary** → models. DTO for HTTP, dao for
  the database, entity for the table.
- **Deciding whether the caller may proceed** → guard. Needs a service or a
  repository to decide, and writes into `ctx.Locals`.
- **Checking that the request is well formed** → middleware. Knows only HTTP.
- **Behaviour that belongs to no domain** → plugin.

The quick rule for a new feature: a table means a **core** module, an HTTP entry
point composing other modules means an **api** module, a stateless helper means
a **utils** module. Details in [module-structure.md](docs/steering/module-structure.md).

## Development Commands

```bash
# Development (hot reload with Air)
make dev [environment]              # Default: development

# Production
make start                          # go run ./cmd/main.go

# Build
make build                          # Output: ./build/auth_service

# Database
make migrate up                     # Run migrations (GORM AutoMigrate)
make migrate down                   # Rollback migrations
make seed init [environment]        # Seed DB with ADMIN app, profiles and user
make seed reset [environment]       # Wipe every seeded table (schema is kept)
make fresh                          # seed reset + migrate + seed init

# Testing
go test ./...                       # Everything
go test ./cmd/                      # FX dependency graph validation
go test ./tests/shared/...          # Repository pattern, dedicated queries, permission algebra
go test ./tests/modules/login/...   # A single module

# Utilities
make cipher <value>                 # Encrypt values for configuration
```

## Directory Organization

```
app/
├── docs/                    # Shared OpenAPI pieces: tags, guard headers, error payloads
├── errors/                  # AppError, error codes, problem details  → docs/steering/error-handling.md
├── middlewares/             # → docs/steering/guards-and-middlewares.md
│   ├── error_handler.go     # The only place that writes an error response
│   ├── validator.go         # BodyValidator[T], Validate[T]
│   ├── guards/              # AppGuard, AuthGuard, OtpGuard, OrganizationGuard, PermissionsGuard
│   └── validators/          # Body validators that switch on a query parameter
└── modules/
    ├── {api}/               # authorize, login, register, healthcheck
    ├── core/                # app, organization, otp, participant, profile,
    │                        # session, user, user_pool
    └── utils/               # cipher, hash, jwt

cmd/
├── main.go                  # appOptions(): the whole FX graph
├── main_test.go             # fx.ValidateApp over that graph
├── database/                # Seeding (init, reset) and migrations
├── helpers/                 # make cipher
└── sandbox/                 # Scratch space, not part of the app

infra/
├── bootstrap/               # Database, logger, http server, email, route guards
├── config/                  # YAML, .env and environment variable loading
└── entities/                # GORM entities, one per table

plugins/                     # Cross cutting fx.Option features  → docs/steering/plugins.md
shared/
├── constants/               # AuthAction, profile keys, ANSI colors
├── email/                   # Email manager and adapters
├── global/                  # Process wide handles
├── interfaces/              # IController, IGuard only
├── models/                  # DTOs shared across modules (RequestInfo)
├── permissions/             # Permission documents, the grant catalog and Resolve  → docs/steering/modules/profiles.md
├── repository/              # Option, BaseRepository, Tx  → docs/steering/repository-pattern.md
└── utils/                   # Cipher, JSON, Pair, random, print helpers

docs/specs/                  # Feature specs. Read the one for a feature before
                             # changing it. 2026-08-23-organizations.md is as-built
                             # and carries the open points; -scoped-profiles.md is
                             # a plan for a future session
docs/steering/                    # Layer documentation
tests/
├── bootstrap/               # Web server behaviour
├── middlewares/             # Guard and middleware behaviour
├── modules/                 # Per module service tests, plus mock/
└── shared/                  # Repository pattern and dedicated query tests
```

## Code Style Conventions

These hold in every layer. Layer specific rules are in the steering documents.

### Naming

- **Interfaces**: prefixed with `I` — `ILoginService`, `IUserRepository`
- **Interface files**: always `interface.go`, singular
- **Receivers**: always `this`
- **Constructors**: always `NewXxx`, returning the concrete type
- **Import aliases**: short, and consistent across the codebase:
  ```go
  import (
      e    "auth_service/app/errors"              // always `e`
      repo "auth_service/shared/repository"       // always `repo`
      dto  "auth_service/app/modules/{module}/models"
      ur   "auth_service/app/modules/core/user/repository"
      us   "auth_service/app/modules/core/user/services"
      entity "auth_service/infra/entities"
  )
  ```
  When a file needs two model packages, prefix the alias: `udto`, `odto`, `sdto`.

### Return values

**At most two return values: the result and an error.** Three or more means the
result has a shape that has not been named yet. Name it and return a pointer:

```go
// no
func (this *RegisterService) ProvisionUser(...) (*entity.User, *entity.Participant, *permissions.Resolved, error)

// yes
type ProvisionedUser struct {
    User        *entity.User
    Participant *entity.Participant
    Permissions *permissions.Resolved
}

func (this *RegisterService) ProvisionUser(...) (*ProvisionedUser, error)
```

Where the struct goes follows [models-layer.md](docs/steering/models-layer.md): a result
shaped by the service, not by the transport, sits next to its contract in
`services/interface.go` — `ProvisionedUser`, `ResolvedParticipation`,
`VerifyConsumableOtpResponse`. `models/` is for what crosses the HTTP and database
boundaries.

The exceptions are the idiomatic Go pairs — `(value, ok)` from a map or a type
assertion, and `(rowsAffected, error)` from a repository.
`SessionService.DecryptSessionToken` predates the rule and is left alone.

### Interface implementation

Every implementation asserts its contract at compile time, including mocks:

```go
var _ IUserService = &UserService{}
var _ ur.IUserRepository = &MockUserRepository{}
```

### Comments

**Keep comments scarce.** The default is no comment: a well named function, a
typed parameter and a small body already say what the code does. A comment that
restates the line below it is noise that goes stale.

Write one only when the *reason* cannot be read from the code:

- a constraint the compiler cannot express (an ordering dependency, a predicate
  that must not be a struct condition)
- a library behaviour that would surprise the next reader (gorm dropping zero
  values, `fx.Invoke` ordering)
- a decision that looks wrong until explained (why a query skips pagination, why
  a value is written after the insert)

Do not write:

- header comments on obvious constructors, getters or DTO fields
- section banners inside a file
- commented-out code — delete it, git has it
- a paragraph of rationale that belongs in a steering document; link the
  document or let it be found there

**Where verbosity is allowed:** infrastructure that is heavily reused, generic,
or reflection driven — `shared/repository/` is the reference. There, a wrong
assumption propagates to every module and the reader cannot see the call sites
from the definition, so explaining the mechanism inline is worth the lines.
`app/errors/` and `app/middlewares/` sit in the same category. Ordinary module
code — controllers, services, models — does not.

When in doubt, put the explanation in the steering document for the layer and
leave the code clean.

### Logging

Zap, with structured fields:

```go
this.logger.Info("User found", zap.Any("user", user))
this.logger.Error("Failed to create session", zap.Error(err))
```

The error handler already logs every error with method, path, host, ip, code and
status — log the internal cause, not the request.

## Multi-tenancy

1. **Applications** — each app has public/secret keys, validated by `AppGuard`
2. **User Pools** — isolated user groups per application
3. **Organizations** — groups of users inside a pool. Every user owns one, and
   `users.current_organization_id` is the scope every listing and mutation is
   filtered by. A user has no permissions of its own: it has permissions *in an
   organization*, through its `Participant` row
4. **Permissions** — enforced by `PermissionsGuard` from
   `organization.profile ∩ participant.profile`, resolved per request by
   `shared/permissions.Resolve`. A profile is written in **grants**
   (`as::users::READ`), which are expanded into route patterns before any of that
   algebra runs; `api` is the administration escape hatch and wins the whole path
   it declares → [profiles.md](docs/steering/modules/profiles.md)

`AppGuard` is mounted by prefix in [infra/bootstrap/routes.go](infra/bootstrap/routes.go)
for `/auth`, `/otp` and `/core` — not by the controllers. The other guards are
chained per route. The full chain, what each guard requires and sets, and how
permission documents are matched are in
[guards-and-middlewares.md](docs/steering/guards-and-middlewares.md).

## Configuration

Three sources, in precedence order:
1. YAML (`.env.{environment}.yaml`)
2. Environment file (`.env.{environment}`)
3. System environment variables

Environment is the first CLI argument, defaulting to `development`:

```bash
go run ./cmd/main.go production
make dev production
```

## Testing

- `tests/shared/` — repository pattern guarantees and per module dedicated
  queries, run against a fake conn pool in gorm `DryRun` mode. No database
  needed.
- `tests/shared/permissions/` — grant expansion, wildcards, `Resolve` and
  `IsSubsetOf`. Pure computation, its own package because `tests/shared/` is
  already `package repository_test`.
- `tests/modules/{module}/` — service tests with testify mocks from
  `tests/modules/mock/`.
- `tests/bootstrap/`, `tests/middlewares/` — web server and middleware behaviour.
- `cmd/main_test.go` — `fx.ValidateApp` type checks the whole container. **Run
  it after changing any constructor signature or FX binding.**

Mocks live in `tests/modules/mock/` and assert their interface with
`var _ IXxx = &MockXxx{}`, so an interface change breaks compilation instead of
failing at runtime inside testify.

## Implementation Notes

1. **Controllers auto-register** through the `group:"controllers"` tag; never
   register one by hand in `main.go`.
2. **`fx.Invoke` order matters** — controllers must register their OpenAPI
   descriptions before the swagger plugin builds the document.
3. **Password security** — argon2 for password hashing.
4. **Token management** — separate access and refresh tokens with configurable
   expiration.
5. **Repository access** — a single optional `repo.Option` (transaction,
   preloads, page, sort); updates go through a pointer based dao.
6. **A unit of work writes through repositories.** A service that owns a
   transaction holds the repositories it writes to, including another module's,
   so every write lands on the same `repo.Option` and one rollback undoes all of
   them. Going through the other module's service is overkill when that method is
   a one to one wrapper, and it is how a write silently escapes the transaction.
   `RegisterService.ProvisionUser` is the reference. **Reads still go through the
   service**, always: they carry rules that must not be duplicated. Full rule and
   the three-case table in [service-layer.md](docs/steering/service-layer.md).
