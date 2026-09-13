# Error Handling

Every error that reaches a caller is an [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807)
problem detail. A handler never writes an error body itself: it returns an
`*errors.AppError` and the central handler renders it.

## The three types

| Type            | Where                          | Role                                                          |
| --------------- | ------------------------------ | ------------------------------------------------------------- |
| `AppError`      | `app/errors/app_error.go`      | What the code throws. Carries title, code, detail and extras.  |
| `ErrorCodePair` | `app/errors/codes.go`          | Binds a machine readable code to an HTTP status.               |
| `ProblemDetail` | `app/errors/problem_detail.go` | The JSON body. Built from an `AppError` by the error handler.  |

`ErrorCodePair` is `utils.Pair[AppErrorCode, int]`: `First` is the code string
the client matches on, `Second` is the HTTP status. Keeping them in one value is
what makes it impossible to throw a code with the wrong status from two
different call sites.

The response body:

```json
{
  "type": "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
  "title": "Conflict",
  "status": 409,
  "detail": "A user with this email already exists in the pool",
  "instance": "/auth/register",
  "code": "USER_ALREADY_EXISTS",
  "data": { "email": "someone@example.com" }
}
```

`code` is the contract with the client — it is stable and machine readable.
`detail` is for humans and may be reworded freely. `data` carries whatever
structured extras were passed.

## Throwing

Import the package as `e` and call a throw helper. Never build an `AppError` by
hand and never call `NewAppError` directly outside `app/errors`.

```go
import e "auth_service/app/errors"

return nil, e.ThrowNotFound("User not found in User Pool")
```

Optional structured extras, which land under `data`:

```go
return e.ThrowBadRequest(
    "Request body could not be parsed",
    utils.JSON{"source": "body", "field": "email"},
)
```

`AppError` implements `error`, so a service returns it as a plain `error` and a
handler returns it straight to fiber.

## Choosing between reusing and creating

**Reuse an existing throw helper** when the client would react to the error the
same way it reacts to the generic one. A caller that gets `NOT_FOUND` on a user
and `NOT_FOUND` on an app behaves identically: it shows "not found". Reuse.

**Create a new code** when the client has to branch on it. `USER_ALREADY_EXISTS`
exists next to `CONFLICT` because the client shows a specific message and offers
a login link; `INVALID_OTP_CODE` exists next to `BAD_REQUEST` because the client
keeps the OTP form open and decrements the attempt counter.

The test is behavioural, not descriptive: *would a client write a different
branch for this?* If the answer is no, the distinction belongs in `detail`, not
in a new code. Codes are a public contract — every one added is one more thing
that can never be renamed.

## Adding a new error

Two steps, both in `app/errors`.

**1. The code pair**, in `codes.go`, under the section of its HTTP status:

```go
// Conflict 409
SessionAlreadyActiveCode ErrorCodePair = ErrorCodePair{
    First:  "SESSION_ALREADY_ACTIVE",
    Second: fiber.StatusConflict,
}
```

`First` is SCREAMING_SNAKE_CASE and describes the situation, not the fix.

**2. The throw helper**, in `app_error.go`, next to the others of the same
status:

```go
func ThrowSessionAlreadyActive(detail string, extra ...utils.JSON) *AppError {
    return NewAppError(
        "Conflict",
        detail,
        SessionAlreadyActiveCode,
        "https://developer.mozilla.org/pt-BR/docs/Web/HTTP/Reference/Status/409",
        extra...,
    )
}
```

The four arguments:

- **title** — the HTTP status name, shared by every error of that status
  (`"Conflict"`, `"Bad Request"`). It is not the place for the specific
  situation; that is `detail`.
- **detail** — passed by the caller, describes this occurrence.
- **code** — the pair from step 1.
- **type** — the MDN page of the status. **It must match the status in the code
  pair.** See the known inconsistency below.

Then document it: any route that can answer the new status needs an
`AddResponse` in its OpenAPI description (see
[controller-layer.md](controller-layer.md)).

## The error handler

`app/middlewares/error_handler.go` is the only place that writes an error
response. It matches in this order, and the order matters:

1. **`*e.AppError`** — everything thrown by the application. Rendered as its
   problem detail.
2. **`*fiber.BindError`** — a payload the binder could not read. Checked before
   `*fiber.Error` because it is the more specific type. `FromBindError` turns it
   into a `BAD_REQUEST` with the source and field under `data`, instead of the
   500 it would otherwise become.
3. **`*fiber.Error`** — errors fiber raises around the handlers: an unmatched
   route, a method a route does not register, a body over `BodyLimit`.
   `FromFiberError` maps the status onto a code pair so the body looks like
   every other error. A 404 on an unmatched route gets a detail naming the
   route the caller asked for.
4. **anything else** — logged as `"Unexpected error"` and answered as a 500 with
   an empty `code`. **An empty `code` in a response means something escaped the
   typed errors.** Treat it as a bug, not as a valid outcome.

Panics never reach this handler as panics: the recoverer installed in
`NewHttpServer` logs the stack and converts them into
`ThrowInternalServerError`.

## Rules

- A service returns `*AppError`, never a raw `errors.New` or a driver error.
  Wrap what comes from below: `return nil, e.ThrowInternalServerError("Failed to
  fetch users")`.
- A controller returns the service error unchanged. Re-wrapping loses the code
  the service chose.
- Never leak an infrastructure message into `detail` on a 500. `err.Error()`
  from gorm can carry a query, a column list or a constraint name.
- Log with the logger, do not put the diagnostic in `detail`. The handler
  already logs every error with method, path, host, ip, code and status.
- Do not import `gorm` to test for `ErrRecordNotFound`. Repositories fold "not
  found" into `nil, nil` — see [repository-pattern.md](repository-pattern.md).

## Known inconsistencies

These are in the code today. Do not copy them into new errors.

- **`ThrowPermissionDeniedError` has a mismatched `type`.**
  `PermissionDeniedErrorCode` answers 403 but the helper passes the MDN page for
  401, so the response says `"status": 403` and points at the 401
  documentation. New helpers must use the page of their own status.
- **`ThrowValidationError` is a duplicate of `ThrowBadRequest`** — same title,
  same `BadRequestCode`, same type. It exists only because the validator
  middleware reads better calling it. Do not add more aliases like this; if two
  helpers share a code they should share a name.
