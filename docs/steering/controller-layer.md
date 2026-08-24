# Controller Layer

A controller is the HTTP adapter of a module: it reads the request, calls one
service method, and writes the response. Every branch it contains that is not
about HTTP belongs in a service.

## When to create one

- Always for an **api module** (`login`, `register`, `authorize`).
- For a **core module** only when the domain is exposed over HTTP (`app`,
  `user`, `user_pool`, `otp`). `session` and `profile` have none — they are
  consumed by other modules, not by clients.
- Never for a **utils module**.

One controller per module. A controller registers every route of its module,
including several groups if needed.

## Anatomy

```go
type AppController struct {
    appService       services.IAppService
    userService      us.IUserService
    authGuard        *guards.AuthGuard
    permissionsGuard *guards.PermissionsGuard
    logger           *zap.Logger

    swagger *openapi.Builder
}

var _ interfaces.IController = &AppController{}

func NewAppController(...) *AppController { ... }
```

- Implements `interfaces.IController`, which is a single `Register(server *fiber.App)`.
- `var _ interfaces.IController = &XxxController{}` is mandatory.
- Guards are injected as concrete pointers (`*guards.AuthGuard`); services are
  injected as interfaces.
- The `*openapi.Builder` is injected too — documentation is written in
  `Register`, not in a separate file.
- Registered in the module with `fx.ResultTags(`group:"controllers"`)` and
  collected in bulk by `main.go`. Never register a controller by hand.

## Handlers

A handler does four things and nothing else:

```go
func (this *AppController) GetApps(ctx fiber.Ctx) error {
    currentUser := ctx.Locals("user").(*entity.User)
    currentApp := ctx.Locals("app").(*entity.App)

    var query dto.GetAppsQuery
    if err := ctx.Bind().Query(&query); err != nil {
        return err
    }

    apps, err := this.appService.FindAll(currentUser, currentApp, &query)

    if err != nil {
        return err
    }

    return ctx.Status(fiber.StatusOK).JSON(apps)
}
```

1. Pull what the guards left in `ctx.Locals`.
2. Bind the payload into a DTO.
3. Call the service.
4. Write the status and the body.

**Return the service error unchanged.** Re-wrapping it with `e.ThrowBadRequest`
throws away the code the service chose. The error handler renders it.

**Return the bind error unchanged too.** `FromBindError` turns a
`*fiber.BindError` into a `BAD_REQUEST` carrying the source and the offending
field, which is strictly better than `e.ThrowBadRequest(err.Error())`. Some
handlers still do the latter — do not copy them.

**`ctx.Locals` assertions are safe only behind the matching guard.**
`ctx.Locals("app")` is set by `AppGuard`, `ctx.Locals("user")` and
`ctx.Locals("session_id")` by `AuthGuard`. A bare type assertion on a route
without the guard panics; the recoverer turns it into a 500, but the route is
still wrong.

## Guards and route registration

`AppGuard` is **not** registered by the controller. It is mounted by prefix in
[infra/bootstrap/routes.go](../infra/bootstrap/routes.go):

```go
server.Use("/auth", appGuard.Act)
server.Use("/otp",  appGuard.Act)
server.Use("/core", appGuard.Act)
```

Anything under those three prefixes already validates the application
credentials. A new route outside them is public unless the controller says
otherwise, and a new prefix that needs application credentials has to be added
there.

The remaining guards are chained per route, in this order:

```go
group.Post(
    "/apps",
    middleware.BodyValidator[dto.CreateAppPayload](),
    this.authGuard.Act,
    this.permissionsGuard.Act,
    this.CreateApp,
)
```

| Guard              | Requires                | Provides in Locals              |
| ------------------ | ----------------------- | -------------------------------- |
| `AppGuard`         | X-Public-Key, X-Pool-Key, X-Secret-Key when private | `app` |
| `AuthGuard`        | `AppGuard` first, Authorization header | `user`, `session_id`, `token_type` |
| `OtpGuard`         | `AppGuard` first, `action` query | delegates to `AuthGuard` for actions that need it |
| `PermissionsGuard` | `AppGuard` and `AuthGuard` first | —                       |

The full behaviour of each guard, and how the permission documents the
`PermissionsGuard` reads are matched, is in
[guards-and-middlewares.md](guards-and-middlewares.md).

`AuthGuard`, `OtpGuard` and `PermissionsGuard` all fail with a 500 if `app` is
missing from `Locals`, and `PermissionsGuard` does the same when `user` is
missing. Those 500s are the diagnostic for a route registered outside the
guarded prefixes, or for a guard chained in the wrong order.

### Validation

`middleware.BodyValidator[T]()` binds the body into `T` and runs the
`go-playground/validator` tags on it, returning a `BAD_REQUEST` with a `fields`
array under `data`. Use it whenever a route takes a body.

When the shape of the body depends on a query parameter, use a validator from
`app/middlewares/validators/` instead — `LoginValidator`, `RegisterValidator`
and `OtpValidator` pick the DTO from `?method=` or `?action=`.

**Known inconsistency:** the validator currently runs *before* the guards, so an
unauthenticated caller can tell a valid payload from an invalid one and the
service does parse work for requests that will be rejected. New routes should
keep following the existing order for consistency, but this is worth revisiting
as a whole rather than route by route.

## Documenting the route

Every route is described in `Register`, right above the registration, using the
constructors in [app/docs](../app/docs). Pick the constructor that matches the
guard chain, so the credentials a caller must send are described identically
everywhere:

| Constructor         | Guard chain                                 |
| ------------------- | -------------------------------------------- |
| `PublicRoute`       | none                                         |
| `AppRoute`          | AppGuard                                     |
| `AuthRoute`         | AppGuard + AuthGuard                         |
| `PermissionedRoute` | AppGuard + AuthGuard + PermissionsGuard      |

Wrap with `docs.Validated(...)` when the route runs a body validator — it adds
the 400 and 422 responses.

```go
this.swagger.Add(
    docs.Validated(
        docs.PermissionedRoute(this.swagger, "POST", "/core/apps", openapi.Options{
            Summary:     "Create an application",
            Description: "Creates an application owned by the authenticated user...",
            Tags:        []string{docs.TagApps},
        }),
    ).
        AddBody(dto.CreateAppPayload{}, openapi.Options{Required: true, Description: "..."}).
        AddResponse(fiber.StatusCreated, entity.App{}, openapi.Options{Description: "..."}),
)
group.Post("/apps", middleware.BodyValidator[dto.CreateAppPayload](), this.authGuard.Act, this.permissionsGuard.Act, this.CreateApp)
```

Notes:

- Paths in the documentation use `{id}`; fiber routes use `:id`. They are
  written twice and are not checked against each other — keep them in sync by
  hand.
- Tags come from the `docs.Tag*` constants. Add a constant rather than a literal.
- Every non standard status the route can answer needs its own `AddResponse`,
  including the ones a new error code introduces.
- A response body that is not an entity or a DTO gets a small unexported struct
  next to the controller whose only purpose is the schema — see
  `getAppResponse` and `healthCheckResponse`.

## Rules

- No business logic. No authorization decisions beyond what the guards do — row
  level visibility is resolved by the service.
- No repository. A controller holding a repository is always a missing service.
- No `*gorm.DB`, no direct SQL, no transaction handling.
- One service call per handler where possible. A handler orchestrating three
  services is a service waiting to be extracted.
- The handler returns `error`; it never writes an error body itself.
