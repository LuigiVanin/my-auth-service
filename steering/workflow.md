# Development Workflow — Methodology

This document is the **method**. The layer rules live in the other steering
documents; what is ordered here is the sequence, and the order is what keeps a
change from being rewritten twice.

For the mandatory checklist before touching anything, see
[CLAUDE.md](../CLAUDE.md). For what to document when, see the hard rule there too.

---

## The five steps, always in this order

```
1. Layer      — where does this change live?
       ↓
2. Contract   — name the shapes before writing the logic
       ↓
3. Rule       — the service decides; the repository only fetches
       ↓
4. Surface    — controller, guard chain, grant, OpenAPI
       ↓
5. Proof      — tests, fx.ValidateApp, migration
```

Skipping a step does not save time, it moves the cost. Writing the handler before
the contract produces a DTO shaped by one screen. Writing the service before
deciding the layer produces query logic inside it, which is the failure this
codebase already lived through once — see the history in
[repository-pattern.md](repository-pattern.md).

---

## Step 1 — Layer

Before opening a file, answer where the change belongs. The table is in
[CLAUDE.md](../CLAUDE.md) under "Which layer am I in?"; the two questions it does
not answer:

- [ ] Does this need a **new module**? A table means a core module, an HTTP entry
      point composing other modules means an api module, a stateless helper means
      a utils module → [module-structure.md](module-structure.md)
- [ ] Does the write belong to a **unit of work**? More than one mutation that has
      to succeed together means a transaction, and inside one every write goes
      through the repository — including another module's →
      [service-layer.md](service-layer.md)

---

## Step 2 — Contract

Name the shapes before writing the logic that fills them.

- [ ] Request body, query and response → `{module}/models/{module}.dto.go`
- [ ] Columns the write is allowed to touch → `{module}/models/{module}.dao.go`,
      every field a pointer
- [ ] A result shaped by the service and not by the transport → next to its
      interface in `services/interface.go`, never in `models/`
- [ ] A column the service owns and no payload may write → out of the dao, into a
      named repository method

Rules and naming: [models-layer.md](models-layer.md).

**The dao is a security boundary, not a convenience.** A column absent from it can
never be written by any request body, whatever the payload says. Deciding what is
in it is a decision about what a caller is allowed to do.

---

## Step 3 — Rule

The service decides what should happen. The repository only knows how to fetch.

- [ ] Query logic stays in the repository — no condition, join or order clause
      built in a service
- [ ] `FindOne` returning `nil, nil` is "does not exist"; whether that is a 404, a
      `false` or a normal branch is a business decision and belongs here
- [ ] Row level visibility is the service's job, not the guard's — the guard
      answers "may this profile call this route", never "which rows may it see"
- [ ] Errors returned to a controller are always typed `*AppError` →
      [error-handling.md](error-handling.md)

**"Not yours" is a 404, never a 403.** Folding it into "does not exist" is what
keeps an id from becoming an oracle for the entities of other organizations. The
exception is a scope error the caller can act on — being in the wrong
organization answers `NOT_A_PARTICIPANT`, because the fix is to switch.

---

## Step 4 — Surface

- [ ] Handler does four things: read `ctx.Locals`, bind, call one service, write
      the status → [controller-layer.md](controller-layer.md)
- [ ] Guard chain in dependency order, and the body validator when there is a body
- [ ] **A new route needs an entry in the grant catalog** or no profile but the
      platform admin reaches it, and nothing checks this automatically →
      [modules/profiles.md](modules/profiles.md)
- [ ] The route is documented in `Register`, with an `AddResponse` for every
      non standard status it can answer
- [ ] **A literal path is registered before a parameter path of the same method**,
      or the parameter route swallows it

---

## Step 5 — Proof

- [ ] `go build ./...`
- [ ] `go vet ./...` — it compiles the tests too, which is how a stale mock or a
      stale constructor call surfaces
- [ ] `go test ./cmd/` after **any** constructor or FX binding change
- [ ] `go test ./...`
- [ ] `go test ./... -race` when the change involves a goroutine
- [ ] Entity changed → `make migrate up`; seed affected → `make fresh`
- [ ] Mocks updated in `tests/modules/mock/` — the `var _ IXxx = &MockXxx{}`
      assertion turns an interface change into a compile error, which is the design

---

## Final checklist before calling it done

- [ ] Build, vet and the whole suite pass
- [ ] The hard rule table in [CLAUDE.md](../CLAUDE.md) was run mentally, and every
      document it points at was updated **in the same change**
- [ ] A feature of real size has its `docs/features/<date-name>/` folder, with
      `requirements.md` written before the code and `implementation-plan.md` grown
      during it
- [ ] Technical debt found outside the scope went to
      [docs/features/pendencias.md](../docs/features/pendencias.md) instead of a
      TODO nobody reads
- [ ] No commit, push or PR without asking — see the Git section in CLAUDE.md

---

## Anti-patterns observed in this codebase

Each of these was a real defect here, not a hypothetical.

### ❌ Answering 404 on `RowsAffected == 0`

> An update dao with every field nil resolves to an empty map, and `Update` short
> circuits to `(0, nil)` without touching the database. A `PUT` with an empty body
> then 404s a row that exists. Ask `repo.HasChanges(dao)` first.

### ❌ Reusing a read whose scope check is not where you think

> `UserService.FindById` returns the caller before checking the pool, so that
> `/core/users/me` works. Reusing it in the write path made self-update
> unrestricted. A write needs its own read.

### ❌ A struct condition with a zero value

> `Find(entity.Session{Invalidated: false})` compiles to no condition at all —
> gorm drops zero valued fields. An invalidated session still authorized. Spell
> the predicate out in a dedicated repository method.

### ❌ A bare `go` for fire and forget

> The fiber recoverer wraps the handler's own stack; a goroutine it spawned
> unwinds alone and takes the process down. Use `utils.Detach`.

### ❌ Capturing, in a goroutine, a pointer the caller goes on to mutate

> Every login track reads the session into a value first, because `session.User`
> is assigned right after. A captured pointer there is a data race the tests only
> find under `-race`.

### ❌ A parameter route registered before the literal it shadows

> `PUT /core/organizations/:id` before `/switch` does not 404 — the
> `PermissionsGuard` matches on `ctx.Route().Path`, so it resolves to the wrong
> permission key and answers 403 to everyone holding the grant of the literal.
> Pinned by `tests/modules/routes/route_order_test.go`.

### ❌ The same document kept in two places

> The seeded permission documents lived in the seed and in the test, and drifted
> by four grants — the test passed while the real profile was refused. One of the
> two has to import the other.

### ❌ A field whose name differs between the payload and the response

> `users` accepted `verify_email` and answered `verifyEmail`. Sending back the
> name you just read was **ignored in silence**: it arrived as "not sent", with no
> 400 and no warning.

### ❌ Documentation written at the end as a report

> A document that lists files created and lines changed ages in one PR and answers
> no future question. `requirements.md` written after the code describes the code,
> not what was asked and decided — and the two diverge exactly where it matters.
