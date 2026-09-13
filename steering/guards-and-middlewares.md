# Guards and Middlewares

Both are `fiber.Handler`s that run before a handler. The split is by
responsibility, and it decides which folder the code goes in.

| | `app/middlewares/` | `app/middlewares/guards/` |
| --- | --- | --- |
| Answers | "is this request well formed?" | "is this caller allowed?" |
| Knows about | HTTP, payloads, errors | applications, users, sessions, profiles |
| Injects | nothing, or a validated payload | data into `ctx.Locals` |
| Depends on | nothing domain shaped | repositories and services |
| Is a struct | no — plain functions and generics | yes — constructor injected |

A handler that needs a repository or a service to decide is a **guard**. A
handler that only inspects the request is a **middleware**.

## The chain

Three places register handlers, in this order at request time.

**1. Global, in [infra/bootstrap/webserver.go](../infra/bootstrap/webserver.go).**
Applied to every request, in registration order:

```go
app.Use(newRecoverer(logger))   // first, so it wraps everything below
app.Use(cors.New(...))
app.Use(middleware.Json)
```

The recoverer is first on purpose: it wraps cors, the guards and every handler.
It logs the panic with a stack trace and converts it into
`ThrowInternalServerError`, so a panic never escapes as a bare 500.

**2. By prefix, in [infra/bootstrap/routes.go](../infra/bootstrap/routes.go).**

```go
server.Use("/auth", appGuard.Act)
server.Use("/otp",  appGuard.Act)
server.Use("/core", appGuard.Act)
```

`AppGuard` is mounted here and **never by a controller**. Everything under those
three prefixes validates the application credentials. A route outside them is
open unless the controller says otherwise; a new prefix that needs application
credentials has to be added to this file.

**3. Per route, in the controller.**

```go
group.Post(
    "/apps",
    middleware.BodyValidator[dto.CreateAppPayload](),
    this.authGuard.Act,
    this.permissionsGuard.Act,
    this.CreateApp,
)
```

The chain is ordered by dependency: each guard reads what the previous one left
in `ctx.Locals`.

## The guards

All four implement `interfaces.IGuard`, a single `Act(ctx fiber.Ctx) error`, and
are provided to the container as concrete pointers in `appOptions()` — not as
interfaces, because a controller injects the specific guard it chains.

### AppGuard

Identifies the calling application. Mounted by prefix, not per route.

| | |
| --- | --- |
| Requires | `X-Public-Key`, `X-Pool-Key`; `X-Secret-Key` when the app is private |
| Sets | `app` (`*entity.App`, with `UsersPool` preloaded), `pool`, and `secretKey` for private apps |
| Fails with | 401 on missing, malformed or mismatched keys; 404 when no app matches |

Both keys are ciphered ids with an `as_` prefix, checked against
`^as_[a-zA-Z0-9._-]+$` before being deciphered. The pool key must belong to the
app the public key resolves to — a valid pool key from another app is a 401.

### AuthGuard

Identifies the user behind the access token.

| | |
| --- | --- |
| Requires | `AppGuard` first; `Authorization` header |
| Sets | `user` (`*entity.User`), `session_id`, `token_type` |
| Fails with | 500 if `app` is missing from `Locals`; 401 on a missing header or a rejected token |

It delegates to `IAuthorizeService.Authorize`, which is where token parsing,
session lookup and IP checking live. The guard itself holds no auth logic.

### OtpGuard

Conditional authentication for the OTP routes.

| | |
| --- | --- |
| Requires | `AppGuard` first; an `action` query parameter |
| Sets | whatever `AuthGuard` sets, when it runs |
| Fails with | 500 if `app` is missing; 400 if `action` is absent |

`REGISTER`, `LOGIN` and `FORGOT_PASSWORD` are reachable without a token — the
caller has no session yet. Every other action falls through to `AuthGuard`.
Adding an action to `constants.AuthAction` does **not** make it unauthenticated;
that list is explicit in the guard.

### PermissionsGuard

Route level authorization from the profile's permission document.

| | |
| --- | --- |
| Requires | `AppGuard` and `AuthGuard` first; `user.Profile` preloaded with `Permissions` |
| Sets | nothing |
| Fails with | 500 if `app` or `user` is missing; 403 on a missing profile, missing permissions, or a denied route, method or query |

Row level visibility is **not** its job. It answers "may this profile call `GET
/core/apps`", not "which apps may this user see" — that decision belongs to the
service. See [service-layer.md](service-layer.md).

Every denial carries the profile's permission document under `data`, so the
caller can see what it was matched against.

## Permission documents

`profiles.permissions` is JSONB, matched by the guard as:

```json
{
  "api": {
    "/core/apps": { "methods": ["POST", "GET"] },
    "/core/apps/:id": { "methods": ["GET", "PUT", "DELETE"] },
    "/core/users": {
      "methods": ["GET"],
      "query": { "app_id": "^.+$", "skip": "^[0-9]+$", "limit": "^[0-9]+$", "name": "^.+$" }
    }
  }
}
```

The ADMIN profile uses the wildcard form: `{"api": {"*": {"methods": ["*"]}}}`, plus a
`"grants": ["as::*::*"]` that the guard never reaches - it matches the `"*"` key first.

Matching rules, in order:

1. `"*"` as a key matches everything.
2. An exact key match against the path.
3. Each key is turned into a regex — `:id` becomes `\d+`, `:uuid` becomes a uuid
   pattern — and tested against the path.

**The path being matched is `ctx.Route().Path`, the registered route pattern, not
the request URL.** A permission key is therefore written as `/core/apps/:id`,
never `/core/apps/9f3c...`. This is the detail that makes step 3 mostly
redundant and the one that breaks a permission document written by hand from a
browser URL.

Methods: `"*"` allows everything, otherwise a case insensitive match.

Query: an absent or empty `query` object allows any query string. `{"*": "*"}`
does the same explicitly. Otherwise **every** query parameter the caller sends
must have a key in the map (or a `"*"` fallback) and its value must match that
key's regex. A parameter the map does not mention is a denial, so adding a query
parameter to a route means updating the profiles that use it.

## Writing a new guard

```go
package guards

var _ i.IGuard = &BillingGuard{}

type BillingGuard struct {
    billingService bs.IBillingService
    logger         *zap.Logger
}

func NewBillingGuard(billingService bs.IBillingService, logger *zap.Logger) *BillingGuard {
    return &BillingGuard{billingService: billingService, logger: logger}
}

func (this *BillingGuard) Act(ctx fiber.Ctx) error {
    app, ok := ctx.Locals("app").(*entity.App)

    if !ok || app == nil {
        return e.ThrowInternalServerError("App context missing in BillingGuard - should be run after AppGuard")
    }

    // ... decide, then:
    return ctx.Next()
}
```

Checklist:

- `var _ i.IGuard = &XxxGuard{}` — with the guard's own type. Copy-pasting a
  neighbouring guard's assertion silently leaves the new one unchecked.
- Read every `Locals` dependency with the two value form and fail with a **500**
  when it is missing. A missing `app` is not a client error, it is a route
  wired without its guard, and the 500 is the diagnostic.
- Depend on **services**, not repositories, when a decision needs data.
  `AppGuard` is the one exception and predates the rule.
- Throw typed errors from `app/errors`; never write a response body.
- Log through the injected `*zap.Logger`, never through `global.Logger`. The
  global exists for code that has no container access, which a guard always has.
- Return `ctx.Next()` on success — a guard that forgets it silently ends the
  chain and answers 200 with an empty body.
- Provide it in `appOptions()` next to the other guards, as a concrete pointer.
- Add the credentials it requires to a constructor in [app/docs](../app/docs) so
  every route behind it documents them identically. See
  [controller-layer.md](controller-layer.md).

## The middlewares

### Error handler

`NewErrorHandler(logger)` is installed as fiber's `ErrorHandler` and is the only
place in the codebase that writes an error response. It is documented in
[error-handling.md](error-handling.md); nothing else should render an error.

### Validators

`middleware.BodyValidator[T]()` binds the body into `T` and runs its `validate`
tags, answering a 400 with a `fields` array under `data`. `middleware.Validate[T]`
is the same check without the binding, for use inside a handler.

When the body shape depends on a query parameter, the route uses a wrapper from
`app/middlewares/validators/` instead:

| Validator | Switches on | Between |
| --- | --- | --- |
| `LoginValidator` | `?method=` | `LoginPayloadWithPassoword`, `LoginPayloadWithOtp` |
| `RegisterValidator` | `?method=` | `RegisterPayloadWithPassoword`, `RegisterPayloadWithOtp` |
| `OtpValidator` | `?action=` | `OtpRegisterPayload`, `OtpLoginPayload` |
| `MethodValidator[P, O]()` | `?method=` | its two type parameters |

`MethodValidator` is the generic form of the first two, which predate it. A new
method switched route should use `MethodValidator`; the existing ones are left
alone because they are referenced from their controllers.

An unknown value answers 422 (`Invalid login method`, `Invalid OTP action`); an
absent one falls back to the password/register variant, except `OtpValidator`
which requires `action` and answers 400 without it.

### Adding a middleware

A middleware is a plain function or a generic returning `fiber.Handler`. It
lives in `app/middlewares/`, imports nothing from `app/modules/`, and is
registered either globally in `NewHttpServer` or per route by a controller.
If it needs a service to decide, it is a guard, not a middleware.

## Accepted deviations

- **`middleware.Json` does nothing observable.** It calls
  `ctx.Accepts("application/json")` and discards the result, so it neither
  negotiates nor enforces the content type. Kept as is deliberately — do not
  "fix" it into an enforcing middleware without deciding what should happen to
  callers that send another content type.
- **`AppGuard` holds `IAppRepository` directly.** Every other guard depends on a
  service. This one predates the rule and is not a precedent: a new guard that
  needs data goes through a service.
