# Module Structure and Wiring

A module is the unit of the application: a folder under `app/modules/`, an
`fx.Module` variable, and the layers it needs. Everything is wired by Uber FX
in [cmd/main.go](../cmd/main.go).

## The three kinds of module

| Kind      | Under                | Has        | Example                        |
| --------- | -------------------- | ---------- | ------------------------------ |
| **api**   | `app/modules/`       | controller, service | `login`, `register`, `authorize` |
| **core**  | `app/modules/core/`  | repository, service, sometimes controller | `user`, `app`, `session`, `otp`, `profile`, `user_pool` |
| **utils** | `app/modules/utils/` | service only, no entity | `cipher`, `hash`, `jwt`         |

Which one to create:

- The feature owns a **database table** → core module. It gets a repository and
  an entity in `infra/entities/`.
- The feature is an **HTTP entry point** composing other modules → api module.
  It has no repository of its own; it depends on core services.
- The feature is a **stateless helper** used by several modules → utils module.
  No entity, no repository.
- The feature is **cross cutting and not domain shaped** (documentation, tracing,
  a dev only endpoint) → not a module. See [plugins.md](plugins.md).

## Anatomy

```
{module}/
├── {module}.module.go             # fx.Module definition
├── controller/
│   └── {module}.controller.go     # HTTP handlers, api modules and exposed core modules
├── models/
│   ├── {module}.dto.go            # request, query and response payloads
│   └── {module}.dao.go            # update daos for the repository
├── services/
│   ├── interface.go               # IXxxService
│   └── {module}.service.go        # business logic
└── repository/                    # core modules only
    ├── interface.go               # IXxxRepository
    └── {module}.repository.go     # data access
```

Not every folder is required — `profile` has no controller and no models,
`healthcheck` is a single file with only a controller. Create a folder when the
layer exists, not preemptively.

The interface file is always `interface.go`, singular. Never `interfaces.go`.

## The module file

```go
package login

var Module = fx.Module(
    "login",

    fx.Provide(
        fx.Private,
        fx.Annotate(
            ur.NewUserRepository,
            fx.As(new(ur.IUserRepository)),
        ),
    ),

    fx.Provide(
        fx.Private,
        fx.Annotate(
            services.NewLoginService,
            fx.As(new(services.ILoginService)),
        ),
    ),

    fx.Provide(
        fx.Annotate(
            controller.NewLoginController,
            fx.As(new(interfaces.IController)),
            fx.ResultTags(`group:"controllers"`),
        ),
    ),
)
```

Three rules hold everywhere:

**Always bind to the interface with `fx.As`.** A constructor returns the
concrete type; the container hands out the interface. This is what lets a test
substitute a mock and what keeps the consumer from reaching past the contract.

**Use `fx.Private` for anything the module does not export.** A repository or a
service that only this module consumes should be private, so another module
cannot accidentally depend on it and so two modules can each build their own
instance. A service other modules consume — `IUserService`, `IOtpService` — is
provided without `fx.Private`.

**Controllers carry `fx.ResultTags(`group:"controllers"`)`.** They are collected
into a slice and registered in bulk in `main.go`; a controller is never
registered by hand.

## Registration order in main.go

`appOptions()` supplies the modules in this order, and it is not cosmetic:

```
infra providers  → config, logger, database, http server, email, transaction manager
guards           → app, auth, otp, permissions
utils modules    → cipher, hash, jwt
core modules     → user, app, user_pool, session, profile, otp
api modules      → register, login, authorize, healthcheck
fx.Invoke        → register every controller in the group
plugins          → swagger
fx.Invoke        → lifecycle hooks, start server
```

`fx.Provide` is lazy, so provider order does not matter. **`fx.Invoke` order
does**: the controllers must register their OpenAPI descriptions before the
swagger plugin builds the document. Adding an `fx.Invoke` between those two
breaks the documentation silently — see [plugins.md](plugins.md).

## Depending on another module

- A service may depend on another module's **service** (`RegisterService` uses
  `IOtpService`, `ISessionService`, `IProfileService`).
- A service must not depend on another module's **repository**. Go through that
  module's service. `AppService` reaching into `IUserPoolRepository` was what
  forced the public key derivation to live inside the repository; it now goes
  through `IUserPoolService`.
- A controller depends on services only, never on a repository.
- A repository depends on nothing but `*gorm.DB`.

## Verifying the wiring

[cmd/main_test.go](../cmd/main_test.go) runs `fx.ValidateApp(appOptions())`,
which type checks every provider and consumer without constructing anything or
touching the database. A constructor signature change that breaks the container
fails there instead of at boot:

```bash
go test ./cmd/
```

Run it after changing any constructor, any `fx.As` binding, or any module list.
