// Package seeds holds the permission documents of the seeded profiles.
//
// They live in a package of their own, and not in cmd/database/init.go, because
// init.go is `package main` and no test can import it - which is how
// tests/shared/permissions came to keep a hand written copy of these documents
// that then drifted from the seed by four grants.
package seeds

import (
	"encoding/json"

	"auth_service/shared/permissions"
)

// A grant is the authoring format: `as::{feature}::{subfeature?}::{ACTION}`,
// translated into route patterns by shared/permissions. Writing api by hand is
// the administration escape hatch, and it is what Admin uses on purpose - see
// below. Every grant here is checked against the catalog by ValidateAll.
var (
	// The "*" of api matches any registered route, catalogued or not, and the guard
	// reads that key first. The grant rides along only so the admin reports a grants
	// list instead of an empty one - it does narrow IsSubsetOf, which prefers an exact
	// key over "*". See steering/modules/profiles.md.
	AdminPermissions = json.RawMessage(`{"api": {"*": {"methods": ["*"]}}, "grants": ["as::*::*"]}`)

	// Carries the whole auth and otp set because it is the ceiling of
	// LoginPermissions, and IsSubsetOf refuses a child that names a route the parent
	// does not.
	ManagerPermissions = json.RawMessage(`{"grants": [
		"as::apps::CREATE", "as::apps::READ", "as::apps::UPDATE",
		"as::users::READ", "as::users::UPDATE",
		"as::users::me::READ", "as::users::me::UPDATE",
		"as::users_pool::CREATE", "as::users_pool::READ", "as::users_pool::UPDATE",
		"as::organizations::CREATE", "as::organizations::READ", "as::organizations::UPDATE",
		"as::organizations::switch::UPDATE", "as::organizations::participants::READ",
		"as::organizations::participants::UPDATE",
		"as::profiles::CREATE", "as::profiles::READ", "as::profiles::UPDATE",
		"as::grants::READ",
		"as::login::CREATE", "as::register::CREATE", "as::authorize::CREATE",
		"as::refresh::CREATE", "as::forgot_password::UPDATE", "as::otp::CREATE"
	]}`)

	// NOTE: nothing enforces /auth yet, so the login and register grants are
	// expressive rather than effective. They are here because the profile a pool
	// defaults to is where "may this pool sign users up" belongs.
	//
	// users::me::UPDATE and not users::UPDATE: editing yourself must not come with
	// editing every user of every pool the organization owns.
	LoginPermissions = json.RawMessage(`{"grants": [
		"as::login::CREATE", "as::register::CREATE",
		"as::users::me::UPDATE",
		"as::organizations::READ", "as::organizations::switch::UPDATE"
	]}`)

	// Nothing assigns it yet; seeded for the invite flow.
	MemberPermissions = json.RawMessage(`{"grants": [
		"as::organizations::READ", "as::organizations::participants::READ"
	]}`)
)

// All is every seeded document, in ceiling order: Admin, then Manager, then the
// two that have to fit under Manager.
func All() []json.RawMessage {
	return []json.RawMessage{
		AdminPermissions,
		ManagerPermissions,
		LoginPermissions,
		MemberPermissions,
	}
}

// ValidateAll fails loudly when a literal above names a grant the catalog does
// not know. Nothing else checks these strings: a key renamed or removed in
// shared/permissions turns them into rows that grant nothing, silently.
func ValidateAll() error {
	for _, document := range All() {
		parsed, err := permissions.Parse(document)

		if err != nil {
			return err
		}

		if err := permissions.ValidateGrants(parsed.Grants); err != nil {
			return err
		}
	}

	return nil
}
