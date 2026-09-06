package permissions_test

import (
	"encoding/json"
	"testing"

	"auth_service/shared/permissions"

	"github.com/stretchr/testify/assert"
)

// A = apps, B = users, C = organizations. Short names so the survival table below
// reads as a table and not as a wall of grant strings.
const (
	grantA = "as::apps::READ"
	grantB = "as::users::READ"
	grantC = "as::organizations::READ"
)

func everyGrant() []string {
	grants := []string{}

	for _, entry := range permissions.Catalog() {
		grants = append(grants, entry.Grants...)
	}

	return grants
}

func resolve(t *testing.T, documents ...json.RawMessage) *permissions.Resolved {
	t.Helper()

	resolved, err := permissions.Resolve(documents...)

	assert.NoError(t, err)

	return resolved
}

func methodsOf(t *testing.T, resolved *permissions.Resolved, path string) []string {
	t.Helper()

	rule, ok := resolved.Api[path]

	assert.True(t, ok, "expected `%s` to be granted", path)

	return rule.Methods
}

func paths(resolved *permissions.Resolved) []string {
	found := []string{}

	for path := range resolved.Api {
		found = append(found, path)
	}

	return found
}

// The whole feature rests on this: a grant is not a new thing the guard has to
// understand, it is a fragment of the api document produced before any algebra
// runs.
func TestGrantExpandsIntoRoutePatterns(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::users::READ"]}`))

	assert.ElementsMatch(t, []string{"/core/users", "/core/users/:id"}, paths(resolved))
	assert.Equal(t, []string{"GET"}, methodsOf(t, resolved, "/core/users"))

	// A grant never restricts a query parameter, and nil is what the guard reads
	// as "any query string".
	assert.Nil(t, resolved.Api["/core/users"].Query)
}

func TestTwoGrantsUnionTheMethodsOfAPath(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::apps::READ", "as::apps::UPDATE"]}`))

	assert.ElementsMatch(t, []string{"GET", "PUT"}, methodsOf(t, resolved, "/core/apps/:id"))
}

// The ACTION of a grant names the intent, not the verb. Deriving the verb from
// the action would drop the PUT here and leave the OTP flow half granted.
func TestOneGrantMayCarryDifferentVerbs(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::otp::CREATE"]}`))

	assert.Equal(t, []string{"POST"}, methodsOf(t, resolved, "/otp/generate_consumable"))
	assert.Equal(t, []string{"PUT"}, methodsOf(t, resolved, "/otp/verify/:otp_id"))
}

// Merging the two sources would open the query of every path a grant also
// reaches, silently erasing the regex someone wrote by hand. api replaces the
// path it declares, and only that path.
func TestApiWinsThePathItDeclares(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{
		"grants": ["as::users::READ"],
		"api": {"/core/users": {"methods": ["GET"], "query": {"skip": "^[0-9]+$"}}}
	}`))

	assert.Equal(t, map[string][]string{"skip": {"^[0-9]+$"}}, resolved.Api["/core/users"].Query)

	// The rest of the grant is untouched.
	assert.Equal(t, []string{"GET"}, methodsOf(t, resolved, "/core/users/:id"))
	assert.Nil(t, resolved.Api["/core/users/:id"].Query)
}

// The action wildcard is what reaches a subfeature, and it stays inside its own
// feature.
func TestActionWildcardReachesTheSubfeaturesOfItsFeature(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::organizations::*"]}`))

	assert.ElementsMatch(t, []string{
		"/core/organizations",
		"/core/organizations/switch",
		"/core/organizations/:id/participants",
		"/core/organizations/:id/participants/:participant_id",
	}, paths(resolved))
}

// A grant without a wildcard is an exact catalog key. Were it a prefix, every
// explicit grant already written would start granting more than it granted.
func TestExactGrantDoesNotReachSubfeatures(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::organizations::READ"]}`))

	assert.ElementsMatch(t, []string{"/core/organizations"}, paths(resolved))
}

// Any wildcard makes the subfeature a don't care, so a feature wildcard with a
// concrete action crosses subfeatures too - `/core/users/me` comes from the
// `as::users::me::READ` key.
func TestFeatureWildcardCrossesSubfeatures(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::*::READ"]}`))

	assert.Contains(t, paths(resolved), "/core/users/me")
	assert.Contains(t, paths(resolved), "/core/organizations/:id/participants")

	// READ only: nothing that the catalog files under another action.
	assert.NotContains(t, paths(resolved), "/auth/login")
}

// The wildcard is bounded by the catalog, not by the route table. That is the
// difference against `api: {"*": {"methods": ["*"]}}`, which matches any
// registered path, catalogued or not.
func TestFullWildcardExpandsToTheWholeCatalog(t *testing.T) {
	whole := resolve(t, json.RawMessage(`{"grants": ["as::*::*"]}`))

	for _, entry := range permissions.Catalog() {
		for _, grant := range entry.Grants {
			single := resolve(t, json.RawMessage(`{"grants": ["`+grant+`"]}`))

			for path := range single.Api {
				assert.Contains(t, paths(whole), path, "`%s` should be inside the full wildcard", grant)
			}
		}
	}

	// Expanded, never as authored: no consumer can expand `as::*::*` without a copy
	// of this catalog.
	assert.ElementsMatch(t, everyGrant(), whole.Grants)
	assert.NotContains(t, whole.Grants, "as::*::*")
}

// The shape that used to report nothing while reaching every route in the service,
// leaving the platform operator looking unprivileged to anything reading grants.
func TestApiOnlyDocumentReportsTheCatalogItReaches(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"api": {"*": {"methods": ["*"]}}}`))

	assert.ElementsMatch(t, everyGrant(), resolved.Grants)

	// And still reaches what the catalog does not name.
	assert.Contains(t, paths(resolved), permissions.Wildcard)
}

// Ignored on read - the guard must never become a 500 because a key left the map
// - and refused on write, where the caller would otherwise believe it granted
// something.
func TestGrantThatMatchesNothingGrantsNothing(t *testing.T) {
	resolved := resolve(t, json.RawMessage(`{"grants": ["as::does_not_exist::*", "as::apps::READ"]}`))

	assert.ElementsMatch(t, []string{"/core/apps", "/core/apps/:id"}, paths(resolved))
	assert.Equal(t, []string{"as::apps::READ"}, resolved.Grants)

	assert.Error(t, permissions.ValidateGrant("as::does_not_exist::*"))
}

// The list handed to a frontend has to be what the user may actually do in the
// organization it is scoped to, so a grant the ceiling does not hold has to
// disappear from it.
func TestResolvedGrantsReportWhatSurvivesTheCeiling(t *testing.T) {
	organization := json.RawMessage(`{"grants": ["` + grantA + `", "` + grantB + `"]}`)

	tests := []struct {
		name         string
		organization json.RawMessage
		participant  json.RawMessage
		expected     []string
	}{
		{
			name:         "the participant holds one grant the ceiling does not",
			organization: organization,
			participant:  json.RawMessage(`{"grants": ["` + grantA + `", "` + grantC + `"]}`),
			expected:     []string{grantA},
		},
		{
			// What the owner of any organization holds. Intersecting the lists
			// would answer [], which is false: the owner holds A and B.
			name:         "a wildcard participant collapses onto the ceiling",
			organization: organization,
			participant:  json.RawMessage(`{"api": {"*": {"methods": ["*"]}}}`),
			expected:     []string{grantA, grantB},
		},
		{
			name:         "the participant holds less than the ceiling",
			organization: organization,
			participant:  json.RawMessage(`{"grants": ["` + grantA + `"]}`),
			expected:     []string{grantA},
		},
		{
			name:         "the participant cannot widen the ceiling",
			organization: json.RawMessage(`{"grants": ["` + grantA + `"]}`),
			participant:  json.RawMessage(`{"grants": ["` + grantA + `", "` + grantB + `"]}`),
			expected:     []string{grantA},
		},
		{
			name:         "a ceiling written in api still lets the grant through",
			organization: json.RawMessage(`{"api": {"/core/apps": {"methods": ["GET"]}, "/core/apps/:id": {"methods": ["GET"]}}}`),
			participant:  json.RawMessage(`{"grants": ["` + grantA + `"]}`),
			expected:     []string{grantA},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := resolve(t, test.organization, test.participant)

			assert.ElementsMatch(t, test.expected, resolved.Grants)
		})
	}
}

// Accepted and documented: the list never promises more than it delivers, and a
// hand written api layer is administration that does not report itself as grants.
// Api stays the truth of what the guard does.
func TestGrantCoveredOnlyInPartDoesNotSurvive(t *testing.T) {
	resolved := resolve(
		t,
		json.RawMessage(`{"api": {"/core/users": {"methods": ["GET"]}}}`),
		json.RawMessage(`{"grants": ["as::users::READ"]}`),
	)

	assert.Empty(t, resolved.Grants)
	assert.Equal(t, []string{"GET"}, methodsOf(t, resolved, "/core/users"))
}

// IsSubsetOf used to iterate childDoc.Api raw, so the grants of a candidate
// profile were never compared against the ceiling: POST /core/organizations and
// POST /core/users_pool accepted a profile_id granting more than the caller held,
// and nothing failed. With profiles written in grants the check was dead letter.
func TestIsSubsetOfSeesTheGrantsOfTheChild(t *testing.T) {
	child := json.RawMessage(`{"grants": ["as::apps::READ"]}`)

	exceeds, err := permissions.IsSubsetOf(child, json.RawMessage(`{"api": {"/core/organizations": {"methods": ["GET"]}}}`))

	assert.NoError(t, err)
	assert.False(t, exceeds)

	within, err := permissions.IsSubsetOf(child, json.RawMessage(`{"grants": ["as::apps::READ", "as::users::READ"]}`))

	assert.NoError(t, err)
	assert.True(t, within)
}

// Mirrors cmd/database/init.go. LOGIN_PROFILE named /auth/login and /auth/register
// while MANAGER_PROFILE did not, so IsSubsetOf refused it and POST /core/users_pool
// answered 403 from any organization whose ceiling was MANAGER - including when no
// default_profile_id was sent, since resolveDefaultProfile checks that branch too.
func TestSeededProfilesFitUnderTheirCeiling(t *testing.T) {
	manager := json.RawMessage(`{"grants": [
		"as::apps::CREATE", "as::apps::READ", "as::apps::UPDATE",
		"as::users::READ", "as::users::me::READ",
		"as::users_pool::CREATE", "as::users_pool::READ",
		"as::organizations::CREATE", "as::organizations::READ",
		"as::organizations::switch::UPDATE", "as::organizations::participants::READ",
		"as::grants::READ",
		"as::login::CREATE", "as::register::CREATE", "as::authorize::CREATE",
		"as::refresh::CREATE", "as::forgot_password::UPDATE", "as::otp::CREATE"
	]}`)

	login := json.RawMessage(`{"grants": [
		"as::login::CREATE", "as::register::CREATE",
		"as::organizations::READ", "as::organizations::switch::UPDATE"
	]}`)

	member := json.RawMessage(`{"grants": [
		"as::organizations::READ", "as::organizations::participants::READ"
	]}`)

	for name, child := range map[string]json.RawMessage{"login": login, "member": member} {
		within, err := permissions.IsSubsetOf(child, manager)

		assert.NoError(t, err)
		assert.True(t, within, "%s should fit under manager", name)
	}

	// ADMIN carries both keys: the "*" of api so an uncatalogued route stays reachable,
	// and as::*::* so the document says in the authoring format what it grants.
	admin := json.RawMessage(`{"api": {"*": {"methods": ["*"]}}, "grants": ["as::*::*"]}`)

	for name, child := range map[string]json.RawMessage{"manager": manager, "login": login, "member": member} {
		within, err := permissions.IsSubsetOf(child, admin)

		assert.NoError(t, err)
		assert.True(t, within, "%s should fit under admin", name)
	}

	resolved := resolve(t, admin, admin)

	assert.ElementsMatch(t, everyGrant(), resolved.Grants)

	// The guard reads the "*" key before any concrete path, so the grants riding along
	// never narrow what the operator may call.
	assert.Contains(t, paths(resolved), permissions.Wildcard)
	assert.Equal(t, []string{"*"}, methodsOf(t, resolved, permissions.Wildcard))
}

// Every document already in the database has no `grants` key. It has to keep
// meaning exactly what it meant.
func TestDocumentWithoutGrantsIsUnchanged(t *testing.T) {
	resolved := resolve(
		t,
		json.RawMessage(`{"api": {"/core/apps": {"methods": ["GET", "POST"], "query": {"skip": "^[0-9]+$"}}}}`),
		json.RawMessage(`{"api": {"*": {"methods": ["*"]}}}`),
	)

	assert.ElementsMatch(t, []string{"GET", "POST"}, methodsOf(t, resolved, "/core/apps"))
	assert.Equal(t, map[string][]string{"skip": {"^[0-9]+$"}}, resolved.Api["/core/apps"].Query)
	assert.Empty(t, resolved.Grants)

	serialized, err := json.Marshal(resolved)

	assert.NoError(t, err)
	assert.NotContains(t, string(serialized), "grants")
}

func TestValidateGrantsRefusesWhatGrantsNothing(t *testing.T) {
	refused := []string{
		"users::READ",              // no namespace
		"other::users::READ",       // not this service
		"as::users::read",          // the action is exact and uppercase
		"as::users::LIST",          // not one of the four actions
		"as::does_not_exist::READ", // well formed, not a key
		"as::does_not_exist::*",    // a wildcard matching nothing
		"as::users::me::*",         // the wildcard form has no subfeature
	}

	for _, grant := range refused {
		assert.Error(t, permissions.ValidateGrant(grant), "`%s` should be refused", grant)
	}

	assert.NoError(t, permissions.ValidateGrants([]string{"as::users::me::READ", "as::*::*", "as::apps::*"}))
}

func TestCatalogIsGroupedAndStable(t *testing.T) {
	entries := permissions.Catalog()

	assert.NotEmpty(t, entries)

	for _, entry := range entries {
		if entry.Feature != "users" || entry.Subfeature != "me" {
			continue
		}

		assert.Equal(t, []string{"READ"}, entry.Actions)
		assert.Equal(t, []string{"as::users::me::READ"}, entry.Grants)

		return
	}

	t.Fatal("the users::me subfeature is missing from the catalog")
}
