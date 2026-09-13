# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

It is an index, not a manual. Each layer has a dedicated document under
`steering/` — read the relevant one **before** writing code in that layer.

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

## Before you start — mandatory checklist

In this order, without skipping:

1. **Read the steering document** for the layer you are about to touch (table below)
2. **Read the method** in [steering/workflow.md](steering/workflow.md) — the five
   steps, and the anti-patterns that were real defects here
3. **If this is a feature of real size** (criterion in the `docs/features/`
   section): create `docs/features/<YYYY-MM-DD-name>/` and write
   `requirements.md` **now**, before any code — not afterwards
4. **Implement**, growing `implementation-plan.md` as each decision is taken — not
   writing it from scratch at the end
5. **Update the documentation** per the hard rule below
6. **Ask before committing** — see the Git section

## Hard rule: keeping the documentation alive

**Every technical change keeps the documentation current in the same change.**
The task is **not done** without the corresponding update.

| When you do this | Update this too |
| --- | --- |
| Add or change an endpoint | [steering/controller-layer.md](steering/controller-layer.md) if the pattern changed, and the OpenAPI block in `Register` always |
| Register a route | The grant catalog in `shared/permissions/grants.go` — a route with no entry is reachable by no profile but the platform admin, and nothing checks this |
| Add a grant to the catalog | The seeded profiles in `cmd/database/seeds/profiles.go`, and [steering/modules/profiles.md](steering/modules/profiles.md) — a new key is reached by every organization owner the moment it lands |
| Add a repository method that is not a generic one | [steering/repository-pattern.md](steering/repository-pattern.md) (dedicated queries) |
| Change how a write reaches the database | [steering/repository-pattern.md](steering/repository-pattern.md) + a DryRun test in `tests/shared/dedicated_queries_test.go` |
| Add a DTO, a dao or an entity field | [steering/models-layer.md](steering/models-layer.md) if it introduces a rule; the entity's `json` tag is a **contract decision**, not a default |
| Add a column the service owns | Keep it out of the update dao, give it a named repository method, and say so in the dao's comment |
| Change a constructor or an FX binding | Run `go test ./cmd/`, and update the mocks in `tests/modules/mock/` |
| Add a guard or a middleware | [steering/guards-and-middlewares.md](steering/guards-and-middlewares.md) |
| Add an error code | [steering/error-handling.md](steering/error-handling.md) + an `AddResponse` on every route that can answer it |
| Change a business rule that crosses modules | [steering/service-layer.md](steering/service-layer.md) |
| Add a cross cutting `fx.Option` | [steering/plugins.md](steering/plugins.md) |
| Change anything the frontend consumes | The handoff in `docs/handoff/` — a field rename is breaking and has to ship on both sides in the same window |
| Find technical debt outside the scope | [docs/features/pendencias.md](docs/features/pendencias.md) — not a `TODO` in the code |
| Deliver a feature of real size | A `docs/features/<YYYY-MM-DD-name>/` folder — see below. **When in doubt, ask first** |

**How it works in practice**: before declaring anything done, run this table
mentally. If a row applies, update it **in the same change**.

## Steering Documents

Detailed conventions live in `steering/`. **Read the document for a layer before
adding to or changing that layer** — each one carries decisions, rules and edge
cases that are not obvious from the code.

| Document | Covers | Read it when |
| -------- | ------ | ------------ |
| [module-structure.md](steering/module-structure.md) | Module anatomy, the api/core/utils split, FX wiring, registration order | Creating a module, wiring a provider, changing a constructor |
| [controller-layer.md](steering/controller-layer.md) | HTTP handlers, guard chains, route registration, OpenAPI docs | Adding or changing an endpoint |
| [guards-and-middlewares.md](steering/guards-and-middlewares.md) | The request chain, the five guards, permission documents, validators | Adding a guard or middleware, changing who may call a route |
| [service-layer.md](steering/service-layer.md) | Business logic, dependency rules, transactions, error wrapping | Adding or changing business rules |
| [repository-pattern.md](steering/repository-pattern.md) | `repo.Option`, `BaseRepository`, update daos, transactions, dedicated queries | Touching anything that reads or writes the database |
| [models-layer.md](steering/models-layer.md) | DTOs, update daos, entities, validation tags, naming | Declaring a payload, a query struct, a dao or an entity |
| [profiles.md](steering/modules/profiles.md) | Permission documents, the pool → organization → participant hierarchy, `permissions.Resolve`, clamp vs refuse | Touching a permission document, a profile, or anything that reads permissions |
| [error-handling.md](steering/error-handling.md) | `AppError`, error codes, problem details, when to add a new error | Throwing, reusing or creating an error |
| [plugins.md](steering/plugins.md) | Cross cutting `fx.Option` features, invoke ordering | Adding behaviour that spans every module |
| [workflow.md](steering/workflow.md) | The five steps, the final checklist, the anti-patterns that were real defects here | **Always**, before starting |

### Which layer am I in?

- **Reading or writing the database** → repository. Query logic never leaves it.
- **Deciding what should happen** → service. Business rules, authorization of
  rows, transactions, orchestration across modules.
- **Writing inside a transaction** → the repository, even another module's.
  Reaching for the service there either drops the write out of the transaction or
  drags `repo.Option` across a service boundary. See
  [service-layer.md](steering/service-layer.md).
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
a **utils** module. Details in [module-structure.md](steering/module-structure.md).

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

## Git

**Ask before `git commit`, `git push` and `gh pr create`.** Every time, including a
commit split into fragments, and most of all before opening a pull request.

A question is not authorization. "Podemos abrir a PR?" asks for an opinion — answer
it and stop. Only an imperative aimed at the action grants permission, and it grants
that action alone: a yes to commit is not a yes to push, and a yes to push is not a
yes to open a PR.

Do the work up to the line and stop there — stage the files, write the commit
message, draft the PR body, run `go build`, `go vet` and the tests — then report
what is ready and wait. Preparing costs nothing and makes the approval a single yes.

A commit rewrites shared history, and a push and a PR are visible to other people
the moment they land. None of the three is cheap to take back, and deciding the work
is ready is the author's call.

## Directory Organization

```
app/
├── docs/                    # Shared OpenAPI pieces: tags, guard headers, error payloads
├── errors/                  # AppError, error codes, problem details  → steering/error-handling.md
├── middlewares/             # → steering/guards-and-middlewares.md
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
│   └── seeds/               # The permission documents of the seeded profiles,
│                            # importable so tests read them instead of copying
├── helpers/                 # make cipher
└── sandbox/                 # Scratch space, not part of the app

infra/
├── bootstrap/               # Database, logger, http server, email, route guards
├── config/                  # YAML, .env and environment variable loading
└── entities/                # GORM entities, one per table

plugins/                     # Cross cutting fx.Option features  → steering/plugins.md
shared/
├── constants/               # AuthAction, profile keys, ANSI colors
├── email/                   # Email manager and adapters
├── global/                  # Process wide handles
├── interfaces/              # IController, IGuard only
├── models/                  # DTOs shared across modules (RequestInfo)
├── permissions/             # Permission documents, the grant catalog and Resolve  → steering/modules/profiles.md
├── repository/              # Option, BaseRepository, Tx  → steering/repository-pattern.md
├── tracking/                # The `tracking` column shape and its arithmetic  → docs/features/2026-09-09-tracking/spec.md
└── utils/                   # Cipher, JSON merge patch, Detach, Pair, random, print

docs/
├── features/                # Feature history — one folder per feature of real
│                            # size, plus pendencias.md. See the section below
└── handoff/                 # Contracts handed to another team (the frontend)
steering/                    # Normative conventions, one per layer
tests/
├── bootstrap/               # Web server behaviour
├── middlewares/             # Guard and middleware behaviour
├── modules/                 # Per module service tests, plus mock/ and routes/
│                            # (route registration order, OpenAPI document build)
└── shared/                  # Repository pattern, dedicated queries, permission
                             # algebra, JSON merge patch, tracking documents
```

## `docs/features/` — feature history and known debt

Two different things live here and do not mix:

| Artifact | Content |
| --- | --- |
| [docs/features/pendencias.md](docs/features/pendencias.md) | The central register of known debt. Describes work **still to do** |
| `docs/features/<YYYY-MM-DD-name>/` | The history of **what was done**, the decisions and their reasons, for one feature of real size |

The date prefix is `YYYY-MM-DD` and not the frontend's `DD-MM-YYYY`: five specs
already used it and it sorts chronologically in a listing.

### When to create a feature folder

**Create `docs/features/<YYYY-MM-DD-name>/` BEFORE implementing, not after.** It is
the first artifact of the task, not the last.

The signal: the task introduces a **new capability** or changes structural
behaviour — a new module, a new route group, a change to how permissions resolve,
a new column the service owns. The practical test: if the task will produce more
than one non obvious decision that someone else, or you in three months, would
have to reconstruct from `git log`, it earns the folder.

**Why before and not after.** `requirements.md` written before any code is what
forces a gap to surface *while it is still cheap*, instead of after it has quietly
become an implicit decision indistinguishable from a real requirement.
Reconstructing the document at the end, from code that already exists, produces a
**description of the code** rather than a record of what was asked and decided —
and those two diverge exactly where documenting matters most.

**Do not create a folder for**: a bug fix, a refactor, a dependency bump, a
migration with no contract change, or anything whose "why" fits in the commit
message. Those leave their trace in the commit, and anything worth remembering
goes as a line in `pendencias.md` or as an edit to the relevant steering document.

**When in doubt, ask.** Creating too many pollutes the history with trivia;
creating too few loses the reason behind decisions that will look arbitrary later.

### What goes in the folder

Two files, each with YAML frontmatter:

| File | When | Content |
| --- | --- | --- |
| `requirements.md` | **Before implementing** — in plan mode, next to the plan. A gap found mid-implementation is edited in **right then**, not deferred | Context, actors, functional requirements (`RF-*`), non functional (`RNF-*`), rules and invariants (`RC-*`), scope in and out, detected gaps, session evidence, sources consulted |
| `implementation-plan.md` | A skeleton before, **grown decision by decision** during, one final pass at the end for what diverged | Objective, decisions taken with the source of each, dependencies, what was deliberately left for later |

Writing both at the end as a retrospective report **does not satisfy the rule**,
even if the text comes out identical. Their value is the job they do *during* the
work — forcing a decision to be explicit, exposing a gap early — not the artifact
left behind.

`requirements.md` frontmatter: `status`, `slug`, `extracted_at`, `gate` (short
category — `new-capability`, `blocker-resolution`, `contract-alignment`),
`modules`, `sources`.

`implementation-plan.md` frontmatter: `status`, `context` (same slug), `created`,
`updated`, `implemented`, `drivers` (one sentence: why this change, what it
unblocks).

**Every decision cites its source** — an explicit request, an answer to
`AskUserQuestion`, a reading of the code or a steering document, or something
discovered during implementation that diverged from the approved plan. Without
that the document is untraceable opinion.

Two folders carry an extra `spec.md`: the as-built narrative of the feature.
Those two were reconstructed after the fact and say so in their frontmatter —
they are the exception the rule above exists to prevent, kept because the history
is worth more than the consistency.

The older features are still loose `.md` files in `docs/features/`, from before
this structure existed. They are as-built specs and stay as they are.

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

Where the struct goes follows [models-layer.md](steering/models-layer.md): a result
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

**Try the three better exits first, in this order.** A comment is the last
resort, not the first:

1. **Rename.** A comment explaining what a value is means the name is wrong.
2. **Extract.** A block that needs a heading comment is a function with a name.
3. **Move it to the steering document.** Rationale that takes a paragraph belongs
   in `steering/`, where it is found by whoever needs it, and not above one
   call site where it rots.

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
- a comment that narrates the writing ("now we…", "note that…")
- the history of a decision. Why the code is like this belongs here; what it used
  to be belongs in git and in the spec

**The budget is four lines, and it is a real ceiling.** A comment longer than that
outside the three packages named below is the signal that the explanation belongs
in a steering document. Move it there and leave one line pointing at it — a
five-line block above a twelve-line function is the failure mode this rule exists
to stop, and `shared/permissions` is where it keeps happening.

A doc comment on an exported symbol is held to the same bar as any other: it earns
its lines by saying something the signature does not.

**Where verbosity is allowed:** infrastructure that is heavily reused, generic,
or reflection driven — `shared/repository/` is the reference. There, a wrong
assumption propagates to every module and the reader cannot see the call sites
from the definition, so explaining the mechanism inline is worth the lines.
`app/errors/` and `app/middlewares/` sit in the same category. Ordinary module
code — controllers, services, models — does not.

When in doubt, put the explanation in the steering document for the layer and
leave the code clean. When in doubt about whether the comment is worth writing at
all, it is not.

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
   it declares → [profiles.md](steering/modules/profiles.md)

`AppGuard` is mounted by prefix in [infra/bootstrap/routes.go](infra/bootstrap/routes.go)
for `/auth`, `/otp` and `/core` — not by the controllers. The other guards are
chained per route. The full chain, what each guard requires and sets, and how
permission documents are matched are in
[guards-and-middlewares.md](steering/guards-and-middlewares.md).

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
   the three-case table in [service-layer.md](steering/service-layer.md).
