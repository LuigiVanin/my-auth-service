# Profiles and Permissions

A profile is a permission document plus a key. The same table serves three
different roles, and which one a row plays is decided by what points at it:

| Pointed at by | Role | Seeded keys |
| --- | --- | --- |
| `users_pool.default_profile_id` | The ceiling handed to every organization born in that pool | `MANAGER_PROFILE`, `LOGIN_PROFILE` |
| `organizations.profile_id` | The ceiling of that organization - nothing inside it may exceed this | `ADMIN`, `MANAGER_PROFILE`, `LOGIN_PROFILE` |
| `participants.profile_id` | What one user holds inside one organization | the organization's own `Admin` profile, or `MEMBER_PROFILE` |

`users.profile_id` does not exist. A user has no permissions of its own: it has
permissions *in an organization*, through its participation.

A profile is either **global** - `organization_id IS NULL`, seeded, visible to every
organization - or **scoped** to one organization, which is then the only one that sees
it and the only one that may apply it. Every read a caller can reach goes through one
choke point, `FindByIdVisibleTo` / `FindVisibleTo`, so a listing can never show a row
a lookup would then refuse. Global means *visible*; whether a profile may be applied
is decided by the comparison against the ceiling, always and separately.

The visibility predicate is raw SQL - `(organization_id IS NULL OR organization_id = ?)` -
because gorm drops a nil pointer from a typed struct condition, which would make the
global half disappear in silence and show every organization every row.

A scoped key is generated, never sent: `<org_uuid>:<NAME_IN_MACRO_CASE>`, built by
`services.ScopedKey`. The uuid prefix is what lets `EDITOR` exist in two organizations
without a key conflict that is not one, and it is why `unique(key)` could stay global.
`Name` is editable and `Key` is not, so after a rename the key still reflects the name
it was born with - it is a handle, not a label.

## The hierarchy

```
users_pool.default_profile   →   organization.profile   →   participant.profile
        (what a new org gets)        (the ceiling)          (what one member holds)
```

Each layer only limits the ones below it. None of them can widen what came
before. This is the single rule the whole model rests on, and it is enforced in
two places, in two different ways.

## Two ways to write a document

`profiles.permissions` accepts two keys, and they say the same thing:

```json
{ "grants": ["as::users::READ", "as::apps::CREATE"] }
```

```json
{ "api": { "/core/users": { "methods": ["GET"], "query": { "skip": "^[0-9]+$" } } } }
```

**Grants are the authoring format** - what a profile editor manipulates, and what
every seeded profile but `ADMIN` is written in. `api` is the administration escape
hatch, for the granularity no grant expresses: a query regex, one specific path. In
practice `api` is empty.

A grant is `as::{feature}::{subfeature?}::(CREATE|READ|UPDATE|DELETE)`, or the
wildcard form `as::({feature}|*)::((ACTION)|*)`. The namespace is always `as`.
The spelling of a feature is the spelling of the resource in the route, underscore
included - `users_pool`, never `usersPool`.

### Grant becomes api before anything else

`shared/permissions/grants.go` holds a static map from grant to a fragment in the
exact shape of `Document.Api`, and `Document.resolved()` expands it **before** any
intersection. That identity is the whole design: `Resolve`, `intersect`,
`IsSubsetOf` and the `PermissionsGuard` operate on route patterns only and never
learn that grants exist.

The map is written by hand rather than derived by convention because a convention
necessarily invents paths that do not exist - `as::apps::UPDATE` would derive
`PUT /core/apps` next to `PUT /core/apps/:id`. At runtime a phantom key is inert,
since the guard only ever looks up `ctx.Route().Path`. But it breaks every
containment comparison: `IsSubsetOf` would refuse a grant because of a route nobody
registered.

**The action names the intent, not the verb.** `as::otp::CREATE` grants a `POST` and
a `PUT`, because generating a code without verifying it grants nothing usable.

### Wildcards

A grant with `*` is a pattern over the **keys of the map**, never over the route
table. It expands to the union of every key it matches.

| Grant | Matches |
| --- | --- |
| `as::organizations::*` | every key of that feature, subfeatures included - `READ`, `switch::UPDATE`, `participants::READ` |
| `as::*::READ` | every `READ` key, in any feature |
| `as::*::*` | the whole map |

A grant **without** a wildcard is an exact key: `as::organizations::READ` grants
`GET /core/organizations` and nothing else. Any wildcard makes the subfeature a
don't care. The asymmetry is deliberate - were an exact grant a prefix, every one
already written would start granting more than it granted.

Because the wildcard is bounded by the map, `as::*::*` is **not** the same as
`api: {"*": {"methods": ["*"]}}`, which matches any registered path, catalogued or
not. That is why `ADMIN` keeps an `api` key: a route added without a map entry has to
stay reachable by the platform operator.

`ADMIN` carries **both** keys — `{"api": {"*": …}, "grants": ["as::*::*"]}`. The `api`
half is what the guard reads, because it looks the `"*"` key up first, so nothing the
operator may call is narrowed. The grant rides along only so the admin reports a
meaningful list instead of an empty one.

The accepted cost is elsewhere: the expansion creates concrete path keys, and
`resolvePath` prefers an exact key over `"*"` — the opposite order from the guard. So
`IsSubsetOf` now refuses a profile naming a route or verb the map does not cover. The
operator can still call such a route; it can no longer delegate it through a profile.

The other side of it: **adding a key to the map is a permission change** for whoever
holds a wildcard, not just the translation of a route. Review a new entry with that
in mind.

### api wins the whole path

Inside one document, an `api` entry **replaces** the expansion for the path it names.
It never merges with it.

```json
{
  "grants": ["as::users::READ"],
  "api": { "/core/users": { "methods": ["GET"], "query": { "skip": "^[0-9]+$" } } }
}
```

`/core/users` is enforced with the query regex; `/core/users/:id`, which the grant
also grants and the `api` entry does not name, still comes from the grant.

Merging would be the literal reading of the word, and it is wrong: a grant always
leaves the query open, `ResolvedRule.Query` can only express "and" between patterns,
so the union would open the query of every path a grant also reaches - silently
erasing a regex someone wrote by hand. That is escalation disguised as convenience.

The consequence, accepted: a path covered by `api` stops following the map. If
`as::users::READ` gains a route tomorrow, a profile that declares `/core/users` in
`api` does not receive it *on that path*.

### Grants never restrict a query parameter

That is what still justifies writing `api` by hand. It also means the guard, which
denies any parameter a document does not mention, is no longer the de facto query
validator for a grant shaped profile: `?limit=abc` now reaches the handler.
Validating a query belongs where validation lives, not in a permission document.

### Adding a route means adding a map entry

A route registered without an entry in `shared/permissions/grants.go` is unreachable
by every grant, wildcard included, and only `ADMIN` gets to it. Nothing checks this
automatically.

The map also covers `/auth` and `/otp`, which no guard enforces today. Those entries
are expressive rather than effective, and they exist because the profile a users pool
defaults to is where "may this pool sign users up" belongs.

## Reading: `permissions.Resolve`

`shared/permissions.Resolve(documents ...json.RawMessage) (*Resolved, error)`
stacks documents from the outermost layer to the innermost and returns what
survives. Arguments are **parent first**:

```go
resolved, err := permissions.Resolve(
    organization.Profile.Permissions,   // the ceiling
    participant.Profile.Permissions,    // what the member holds
)
```

It lives in `shared/`, not in the profile module, because it has no dependencies
and every layer needs it: the guard enforces it, the services report it, and a
future profile endpoint will clamp with it. Making it a service method would
force a DI edge for pure computation over two JSON values.

**This is the only correct answer to "what may this caller do".** A participant
profile read on its own overstates whenever the ceiling above it is narrower: a
`MEMBER_PROFILE` participation says "list organizations", but in an organization
whose ceiling does not reach that route the member may not. Only the resolved
document is the truth, and it is what the guard enforces.

Resolution rules, in order:

1. **Paths** — a path survives only when it matches in *both* documents. A
   concrete key of one side is still a candidate when the other side only carries
   `"*"`, which is what makes a `"*"` participant collapse onto the ceiling
   instead of widening it. The match is exact, never a regex: both documents are
   keyed by registered route patterns, so there is nothing to normalise between
   them.
2. **Methods** — the intersection of the two lists. `"*"` on one side yields the
   other side's list; `"*"` on both yields `["*"]`. A path whose methods cancel
   out is dropped rather than kept as an entry that can only deny.
3. **Query** — an absent or empty object allows any query string, and so does
   `{"*": "*"}`. Otherwise every constraint of both sides is kept.

### What `Resolved.Grants` reports

`Resolve` answers the same question by resource, and that list is what a frontend
renders. The rule is **not** the intersection of the declared lists: an organization
of `[A, B]` under a participant of `{"api": {"*": {"methods": ["*"]}}}` - what the
owner of any organization holds - really does grant A and B, and intersecting the
lists would answer `[]`.

> **candidates** = the union of the grants declared by every layer.
> **surviving** = the grant whose whole expansion fits inside the final
> `resolved.Api`.

On documents written entirely in grants this is exactly the intersection of the
lists. Containment is what also answers correctly when one of the layers is written
in `api`.

A grant covered only in part does not appear. If a ceiling written by hand grants
`GET /core/users` but not `GET /core/users/:id`, `as::users::READ` is not reported,
even though `resolved.Api` keeps granting the listing. `Api` is the truth of what the
guard does; `Grants` is an honest summary by resource, and it never promises more than
it delivers. Whoever writes `api` by hand is using the administration escape hatch,
and that does not report itself as grants.

A wildcard is reported as it was declared, never expanded, so it survives whole or
not at all.

### Why a resolved query key is a list

`Resolved.Query` is `map[string][]string`, not `map[string]string`. Two layers can
constrain the same parameter with different regexes, and Go's RE2 has no
lookahead, so there is no single expression meaning "matches both". Keeping one
side and discarding the other would grant more than that other side allows, which
is an escalation. Every pattern is kept and the guard requires all of them to
match.

That is why `Resolved` is a distinct type from `Document` and not just a
normalised `Document`.

## Writing: always refuse

A permission document reaches the database in two ways — picking an existing profile
by id, or authoring a new one — and **both are refused when they exceed what the
caller holds**. Nothing is narrowed silently on write.

The clamp still exists, but only at read time: `Resolve` runs on every request and is
the safety net for a row written under a ceiling that has since narrowed.

### Picking an existing profile as a ceiling

Three routes take a profile chosen by name, and the request is **refused** with
`PERMISSION_DENIED` when the chosen profile grants more than whoever is choosing may
hand out:

| Route | Field | Checked against |
| --- | --- | --- |
| `POST /core/users_pool` | `default_profile_id` | what the **caller** holds in its organization — `Resolve(ceiling, participation)`, then `IsWithin` |
| `PUT /core/organizations/:id/participants/:participant_id` | `profile_id` | the same |
| `POST /core/organizations` | `profile_id` | the **ceiling of the organization**, with `IsSubsetOf` |

The last one is the odd one out, and knowingly: closing it the same way would mean a
narrow participant delegated "create organization" could only ever produce narrow
organizations. It is a product decision, deferred — open point 11 in
[2026-08-23-scoped-profiles.md](../../specs/2026-08-23-scoped-profiles.md). Whoever
picks it up has to look at the branch where `profile_id` is omitted at the same time:
`resolveCeiling` returns the default profile of the pool with an early `return` and
checks nothing at all, which is the only one of the three with an unchecked path.

The difference between "the ceiling of the organization" and "what the caller holds"
only became visible once participants could be narrower than their organization,
which is what scoped profiles introduced.

Refusing rather than clamping is deliberate here: the row would still be labelled
`ADMIN` while behaving like something narrower, and a caller reading
`profile.key` afterwards would be misled.

**A caller always names a profile by id, never by key.** A key is a seed handle: it
is how `cmd/database/init.go` stays idempotent and how a human recognises a row. The
one exception is `UserPoolService.resolveDefaultProfile`, which resolves
`LOGIN_PROFILE` by key when a pool is created without a ceiling — the only profile
the code has to name with no id in hand. If a second `FindByKey` call site appears,
it is worth asking why.

### Authoring a document

`POST /core/profiles` and `PUT /core/profiles/:id` refuse alike. A check only on
create would be no check at all — you would author a bounded document and then widen
it with a `PUT`.

The body carries **grants only**, as an object rather than a bare list, so the day
`api` becomes writable it is a second key of the same object and nobody has to
change. An `api` key sent today is dropped in silence by `json.Unmarshal`, and the
response returns the row as it was written, so the caller can see it was not applied.

The `PUT` writes the `grants` half and carries the stored `api` half over untouched.
Its jurisdiction is the half it speaks; writing the payload whole would erase, in
silence, the granularity no grant expresses. The accepted consequence is that the
`api` half of a row can come to exceed the ceiling without the API noticing — safe at
runtime, because `Resolve` clamps every request anyway, but the row stops being a
promise about what it grants, in that half.

The ceiling of whoever is writing is the whole chain above them, resolved once:

```go
ceiling, err := permissions.Resolve(
    organization.Profile.Permissions,   // the ceiling of the organization
    participant.Profile.Permissions,    // what the caller holds in it
)

within, err := permissions.IsWithin(payload.Permissions, ceiling)
```

The participant layer is not optional. Checking only against the organization's
ceiling would let a member author a profile wider than the member itself holds.

`IsWithin(child json.RawMessage, parent *Resolved) (bool, error)` is the `withinApi`
helper that `IsSubsetOf` is built on, exported with a `*Resolved` parent instead of a
raw one.

**Never compare grant lists instead.** It is the obvious simplification now that only
grants are written, and it breaks the most powerful caller: the platform admin holds
`Resolved.Grants == []`, because `ADMIN` is written in `api` and no layer of it
declares a grant. Comparing lists would leave whoever may do everything unable to
author anything. The comparison is always against the resolved `Api`.

**What gets written is the payload, verbatim.** That is the practical gain of
refusing rather than clamping: there is no resolved document to convert back before
writing, so `Resolved.Query` being `map[string][]string` while `Document.Query` is
`map[string]string` never becomes a problem.

On update, a nil `Permissions` touches nothing. A filled one is checked against the
ceiling **as it is now**, so a document that was legal when the row was created is
refused today if the ceiling narrowed in between. That is intended — nothing may hold
more than its ceiling grants.

`as::*::*` under a ceiling that does not reach the whole catalog is refused whole. It
is never written in part.

### Who may write whose permissions

Refusing what exceeds the caller is only half of it. The other half is that nobody
may raise themselves, and that needs to look at *whose* profile is being touched, not
only at the document:

| Rule | Where |
| --- | --- |
| A global profile is never editable | `ProfileService.writable` |
| The `<org_id>:ADMIN` profile is never editable | `ProfileService.writable` |
| Nobody edits the profile it participates with | `ProfileService.writable` |
| Nobody creates or edits a profile wider than it holds | `ProfileService.documentWithin` |
| Nobody moves its own participation | `OrganizationService.UpdateParticipant` |
| The participation of the owner is frozen | `OrganizationService.UpdateParticipant` |
| Nobody hands another a profile wider than it holds | `OrganizationService.UpdateParticipant` |

The last one is the least obvious and the reason the previous six are not enough:
without it, two narrow participants who may both change participants promote each
other and end up holding everything their organization holds.

Freezing the owner's participation has a side effect that is a guarantee rather than
an accident — every organization keeps at least one full administrator who cannot be
demoted, so nobody can lock everyone out of an organization. Its cost is that there
is no way to transfer one; that needs a flow of its own, moving
`organizations.owner_user_id` and the participation in the same transaction.

The escalation paths each rule closes are in
[2026-08-23-scoped-profiles.md, Fase 8](../../specs/2026-08-23-scoped-profiles.md#fase-8--quem-pode-escrever-permissão-de-quem).

## What is exposed

A raw profile document is misleading on its own, so responses carry resolved ones.

- `entity.Organization.Profile` is `json:"-"`. The ceiling of an organization is
  never serialized.
- `entity.Participant.Profile` is serialized, because the role is meaningful
  metadata, but the permissions inside it are the un-resolved ones.
- Single-user answers (login, register, refresh) use
  `models.UserResponse.Profile`, a `ProfileResponse` that embeds the participant
  profile and **shadows** its `Permissions` with the resolved document. So
  `user.profile.permissions` is always what the caller may actually do, and
  `user.profile.permissions.grants` is that same answer by resource - the list a
  frontend renders.

`GET /core/grants` reports the map itself, grouped by feature, plus the wildcard
forms, which are not map keys and so cannot be enumerated. It exists so a profile
editor does not hardcode a list that diverges on the first new resource.

Listings do not resolve. `GET /core/organizations` and `GET /core/apps` report no
permissions at all.

`GET /core/profiles` is the deliberate exception: there the profile *is* the subject
rather than a layer under something, and whoever is choosing which one to apply has
to see the document as it stands. It reports the global rows plus the ones scoped to
the current organization; `organization_id` can only narrow that page, never widen
it, so any other id answers an empty page rather than a window into another
organization.

## The seeded profiles

Created by `cmd/database/init.go`. Keys live in `shared/constants/profile.go` -
compare against those constants, never against a literal. They exist for the seed
and for `resolveDefaultProfile`; they are not part of any request or response
contract.

| Key | Written in | Grants |
| --- | --- | --- |
| `ADMIN` | both | `{"api": {"*": …}, "grants": ["as::*::*"]}` - the platform administrator's organization only, and the one seeded profile that is **scoped**: the last step of the seed sets its `organization_id` to `admin_organization`. The two keys are deliberate: see the Wildcards section |
| `MANAGER_PROFILE` | grants | apps, users pools, organizations, participants and profiles - an organization that builds the platform out. Carries the whole `/auth` and `/otp` set because it is the ceiling of `LOGIN_PROFILE`, and `IsSubsetOf` refuses a child naming a route the parent does not |
| `LOGIN_PROFILE` | grants | list your organizations and switch between them, plus login and signup; the default for any pool created through the API, so a new pool is born closed |
| `MEMBER_PROFILE` | grants | read only; nothing assigns it yet, it is seeded for the invite flow |

Adding a route means adding a map entry, and then updating every profile that should
reach it. The guard still denies by default; it no longer constrains query
parameters for these profiles, because grants leave the query open.

`cmd/database/init.go` runs `permissions.ValidateGrants` over these literals before
seeding, so a key renamed or removed in the map fails the seed instead of quietly
producing rows that grant nothing.

Permission keys are matched against `ctx.Route().Path` - the registered route
pattern - so a key is written `/core/apps/:id`, never `/core/apps/9f3c...`.

## Who owns an organization, and what they hold

Ownership is `organizations.owner_user_id`, and nothing else. Every organization is
born with an `Admin` profile **scoped to itself**, key `<org_uuid>:ADMIN`, document
`{"grants": ["as::*::*"]}`, and the owner participates on that. Both paths that create
an organization write it inside their existing transaction, through
`IProfileRepository` rather than through the profile service, so one rollback undoes
the organization and its profile together:

- `RegisterService.ProvisionUser` - signup
- `OrganizationService.CreateForUser` - `POST /core/organizations`

`users_pool.default_profile_id` is still the single decision about what a signup gets:
it is the **ceiling** of the new organization, and a wildcard participation under it
resolves back onto it. `Resolve(ceiling, "*")` collapses to the ceiling - that is what
`candidatePaths` is for - so the owner holds the most that organization can hold and
not a token more.

**Why a wildcard and not a copy of the ceiling.** A copy is a snapshot: raise the
ceiling later and the owner does not follow. The wildcard tracks it forever, with no
drift and nothing to keep in sync. It answers, for the owner only, the long standing
complaint that a participation points at a profile *row* rather than at "whatever the
ceiling happens to be" - every other participant is still pinned.

**Why it reads well now, when a wildcard participation was tried and dropped before.**
Two things changed. The row has an identity - what is shown is a profile named `Admin`
scoped to an organization, not a loose document. And `Resolved.Grants` keeps only the
grants whose whole expansion fits the resolved api, so a wildcard under a narrow
ceiling reports the concrete grants of that ceiling. An owner with `as::*::*` under a
`LOGIN_PROFILE` ceiling reports the four `LOGIN_PROFILE` grants, not `["as::*::*"]`.
The raw wildcard only surfaces in the participant listing, and there it comes with a
name.

`admin_organization` is the exception: its owner keeps participating on the `ADMIN`
row itself. `as::*::*` is bounded by the catalog while `api: {"*"}` matches any
registered route, and the two only diverge above the catalog - which is exactly the
reach `ADMIN` exists to keep. A grant shaped `Admin` would clamp the platform operator
to the catalog.

The `Admin` row is refused by `ProfileService.writable`, so it stays a wildcard. There
is no "admin of this organization minus apps": that is a new profile, assigned to
somebody.

**The invariant this closes:** `participants.profile_id` always points at a profile
visible to that organization - scoped to it, or global. `FindByIdVisibleTo` closes the
ids that arrive in a payload; this closes the ones the code writes itself. An invite
endpoint will be the next place to have to respect it.
