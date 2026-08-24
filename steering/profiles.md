# Profiles and Permissions

A profile is a permission document plus a key. The same table serves three
different roles, and which one a row plays is decided by what points at it:

| Pointed at by | Role | Seeded keys |
| --- | --- | --- |
| `users_pool.default_profile_id` | The ceiling handed to every organization born in that pool | `MANAGER_PROFILE`, `LOGIN_PROFILE` |
| `organizations.profile_id` | The ceiling of that organization - nothing inside it may exceed this | `ADMIN`, `MANAGER_PROFILE`, `LOGIN_PROFILE` |
| `participants.profile_id` | What one user holds inside one organization | the organization's own ceiling, or `MEMBER_PROFILE` |

`users.profile_id` does not exist. A user has no permissions of its own: it has
permissions *in an organization*, through its participation.

## The hierarchy

```
users_pool.default_profile   →   organization.profile   →   participant.profile
        (what a new org gets)        (the ceiling)          (what one member holds)
```

Each layer only limits the ones below it. None of them can widen what came
before. This is the single rule the whole model rests on, and it is enforced in
two places, in two different ways.

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

### Why a resolved query key is a list

`Resolved.Query` is `map[string][]string`, not `map[string]string`. Two layers can
constrain the same parameter with different regexes, and Go's RE2 has no
lookahead, so there is no single expression meaning "matches both". Keeping one
side and discarding the other would grant more than that other side allows, which
is an escalation. Every pattern is kept and the guard requires all of them to
match.

That is why `Resolved` is a distinct type from `Document` and not just a
normalised `Document`.

## Writing: clamp, or refuse

There are two ways a permission document reaches the database, and they are
handled differently on purpose.

### Picking an existing profile as a ceiling — refuse

`POST /core/users_pool` and `POST /core/organizations` take a `*_profile_id`. If
that profile grants more than the ceiling of the organization making the request,
the request is **refused** with `PERMISSION_DENIED`, using
`permissions.IsSubsetOf(requested, granterCeiling)`.

Refusing rather than clamping is deliberate here: the row would still be labelled
`ADMIN` while behaving like something narrower, and a caller reading
`profile.key` afterwards would be misled.

**A caller always names a profile by id, never by key.** A key is a seed handle: it
is how `cmd/database/init.go` stays idempotent and how a human recognises a row. The
one exception is `UserPoolService.resolveDefaultProfile`, which resolves
`LOGIN_PROFILE` by key when a pool is created without a ceiling — the only profile
the code has to name with no id in hand. If a second `FindByKey` call site appears,
it is worth asking why.

### Authoring a document — clamp

**There is no endpoint that creates a profile yet.** When one lands, it must not
refuse: it must clamp. The permissions written are
`permissions.Resolve(granterCeiling, requested)`, so a caller can never author a
document that exceeds what its own organization holds, whatever it sends.

The distinction is: picking a named thing that does not fit is a mistake worth
reporting; authoring a document is a request to be bounded.

## What is exposed

A raw profile document is misleading on its own, so responses carry resolved ones.

- `entity.Organization.Profile` is `json:"-"`. The ceiling of an organization is
  never serialized.
- `entity.Participant.Profile` is serialized, because the role is meaningful
  metadata, but the permissions inside it are the un-resolved ones.
- Single-user answers (login, register, refresh) use
  `models.UserResponse.Profile`, a `ProfileResponse` that embeds the participant
  profile and **shadows** its `Permissions` with the resolved document. So
  `user.profile.permissions` is always what the caller may actually do.

Listings do not resolve. `GET /core/organizations` and `GET /core/apps` report no
permissions at all.

## The seeded profiles

Created by `cmd/database/init.go`. Keys live in `shared/constants/profile.go` -
compare against those constants, never against a literal. They exist for the seed
and for `resolveDefaultProfile`; they are not part of any request or response
contract.

| Key | Grants |
| --- | --- |
| `ADMIN` | `{"api": {"*": {"methods": ["*"]}}}` - the platform administrator's organization only |
| `MANAGER_PROFILE` | apps, users pools and organizations - an organization that builds the platform out |
| `LOGIN_PROFILE` | list your organizations and switch between them; the default for any pool created through the API, so a new pool is born closed |
| `MEMBER_PROFILE` | read only; nothing assigns it yet, it is seeded for the invite flow |

Adding a route means updating every profile that should reach it. The guard denies
by default, and it denies **any query parameter the document does not mention**,
so a new query parameter also means touching the profiles.

Permission keys are matched against `ctx.Route().Path` - the registered route
pattern - so a key is written `/core/apps/:id`, never `/core/apps/9f3c...`.

## Who owns an organization, and what they hold

Ownership is `organizations.owner_user_id`, and nothing else. There is no "owner
profile": the owner participates on **the organization's own ceiling**, so it holds
the most that organization can hold and not a token more.

That is what makes `users_pool.default_profile_id` the single decision about what a
signup gets. The pool hands a ceiling to the new organization, and the owner
participates on that same row - one value to configure instead of two rows that
have to agree.

**Consequence worth knowing:** the participation points at a profile *row*, not at
"whatever the ceiling happens to be". Raising the ceiling of an organization later
does not raise its owner, because the participation still points at the old row.
Whoever writes that flow has to move the participation too. An earlier design used
a wildcard participation, which tracked the ceiling automatically but read as
unlimited everywhere it was shown.
