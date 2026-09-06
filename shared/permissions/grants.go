package permissions

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// A grant is translated into a fragment of Document.Api before any algebra runs, so
// nothing downstream knows grants exist.
//
//	as::{feature}::{subfeature?}::(CREATE|READ|UPDATE|DELETE)   exact
//	as::({feature}|*)::((ACTION)|*)                             wildcard
const (
	grantNamespace = "as"
	grantSeparator = "::"
)

var (
	exactGrant    = regexp.MustCompile(`^` + grantNamespace + `::[a-z0-9_]+(::[a-z0-9_]+)?::(CREATE|READ|UPDATE|DELETE)$`)
	wildcardGrant = regexp.MustCompile(`^` + grantNamespace + `::([a-z0-9_]+|\*)::(CREATE|READ|UPDATE|DELETE|\*)$`)
)

// Every value is a fragment in the shape of Document.Api, and an absent Query means
// open: a grant never restricts a query parameter.
//
// The ACTION names the intent, not the verb - as::otp::CREATE grants a POST and a PUT.
//
// A key may only name a registered route, and a route with no key here is reachable
// by no grant, wildcard included. The /auth and /otp keys are not enforced by
// anything today; they exist so a users pool profile can express them.
var catalog = map[string]map[string]Rule{
	"as::apps::CREATE": {
		"/core/apps": {Methods: []string{"POST"}},
	},
	"as::apps::READ": {
		"/core/apps":     {Methods: []string{"GET"}},
		"/core/apps/:id": {Methods: []string{"GET"}},
	},
	"as::apps::UPDATE": {
		"/core/apps/:id": {Methods: []string{"PUT"}},
	},

	"as::users::READ": {
		"/core/users":     {Methods: []string{"GET"}},
		"/core/users/:id": {Methods: []string{"GET"}},
	},
	"as::users::me::READ": {
		"/core/users/me": {Methods: []string{"GET"}},
	},

	"as::users_pool::CREATE": {
		"/core/users_pool": {Methods: []string{"POST"}},
	},
	"as::users_pool::READ": {
		"/core/users_pool":     {Methods: []string{"GET"}},
		"/core/users_pool/:id": {Methods: []string{"GET"}},
	},

	"as::organizations::CREATE": {
		"/core/organizations": {Methods: []string{"POST"}},
	},
	"as::organizations::READ": {
		"/core/organizations": {Methods: []string{"GET"}},
	},
	"as::organizations::switch::UPDATE": {
		"/core/organizations/switch": {Methods: []string{"PUT"}},
	},
	"as::organizations::participants::READ": {
		"/core/organizations/:id/participants": {Methods: []string{"GET"}},
	},
	"as::organizations::participants::UPDATE": {
		"/core/organizations/:id/participants/:participant_id": {Methods: []string{"PUT"}},
	},

	"as::profiles::CREATE": {
		"/core/profiles": {Methods: []string{"POST"}},
	},
	"as::profiles::READ": {
		"/core/profiles":     {Methods: []string{"GET"}},
		"/core/profiles/:id": {Methods: []string{"GET"}},
	},
	"as::profiles::UPDATE": {
		"/core/profiles/:id": {Methods: []string{"PUT"}},
	},

	"as::grants::READ": {
		"/core/grants": {Methods: []string{"GET"}},
	},

	"as::login::CREATE": {
		"/auth/login": {Methods: []string{"POST"}},
	},
	"as::register::CREATE": {
		"/auth/register": {Methods: []string{"POST"}},
	},
	"as::authorize::CREATE": {
		"/auth/authorize": {Methods: []string{"POST"}},
	},
	"as::refresh::CREATE": {
		"/auth/refresh": {Methods: []string{"POST"}},
	},
	"as::forgot_password::UPDATE": {
		"/auth/forgot_password": {Methods: []string{"PUT"}},
	},

	"as::otp::CREATE": {
		"/otp/generate_consumable": {Methods: []string{"POST"}},
		"/otp/verify/:otp_id":      {Methods: []string{"PUT"}},
	},
}

type CatalogEntry struct {
	Feature    string   `json:"feature"`
	Subfeature string   `json:"subfeature,omitempty"`
	Actions    []string `json:"actions"`
	Grants     []string `json:"grants"`
}

// An exact grant is a catalog key or it is nothing; a wildcard is a pattern over
// (feature, action) with the subfeature a don't care. Reading and writing both come
// through here so the rule cannot drift between them.
func matchCatalog(grant string) []string {
	if !strings.Contains(grant, Wildcard) {
		if !exactGrant.MatchString(grant) {
			return nil
		}

		if _, ok := catalog[grant]; !ok {
			return nil
		}

		return []string{grant}
	}

	pattern := wildcardGrant.FindStringSubmatch(grant)

	if pattern == nil {
		return nil
	}

	feature, action := pattern[1], pattern[2]
	keys := []string{}

	for key := range catalog {
		keyFeature, _, keyAction := splitGrant(key)

		if feature != Wildcard && feature != keyFeature {
			continue
		}

		if action != Wildcard && action != keyAction {
			continue
		}

		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

// An unknown or unmatched grant is ignored rather than raised: the guard must not
// become a 500 because a key left the map, and ignoring grants nothing.
func expandGrants(grants []string) map[string]ResolvedRule {
	expanded := map[string]ResolvedRule{}

	for _, grant := range grants {
		for _, key := range matchCatalog(grant) {
			for path, rule := range catalog[key] {
				current, ok := expanded[path]

				if !ok {
					// The catalog is package state and Document.resolved hands these
					// slices out without copying them.
					expanded[path] = ResolvedRule{
						Methods: slices.Clone(rule.Methods),
						Query:   singlePatterns(rule.Query),
					}

					continue
				}

				expanded[path] = ResolvedRule{
					Methods: unionMethods(current.Methods, rule.Methods),
					Query:   current.Query,
				}
			}
		}
	}

	return expanded
}

// The candidates Resolve filters against the resolved api.
func knownGrants(grants []string) []string {
	known := []string{}

	for _, grant := range grants {
		if len(matchCatalog(grant)) == 0 || slices.Contains(known, grant) {
			continue
		}

		known = append(known, grant)
	}

	slices.Sort(known)

	return known
}

// A well formed grant that reaches no key is refused too, because the alternative is
// a caller believing it granted something. Returns a plain error: this package
// depends on nothing, so wrapping it into a 400 is the caller's job.
func ValidateGrant(grant string) error {
	if len(matchCatalog(grant)) == 0 {
		return fmt.Errorf("`%s` is not a grant this service knows", grant)
	}

	return nil
}

func ValidateGrants(grants []string) error {
	for _, grant := range grants {
		if err := ValidateGrant(grant); err != nil {
			return err
		}
	}

	return nil
}

// Grouped and sorted here rather than in the controller: it is knowledge of the
// grant grammar, not of the transport.
func Catalog() []CatalogEntry {
	grouped := map[string]*CatalogEntry{}

	for grant := range catalog {
		feature, subfeature, action := splitGrant(grant)
		key := feature + grantSeparator + subfeature

		entry, ok := grouped[key]

		if !ok {
			entry = &CatalogEntry{Feature: feature, Subfeature: subfeature}
			grouped[key] = entry
		}

		entry.Actions = append(entry.Actions, action)
		entry.Grants = append(entry.Grants, grant)
	}

	entries := make([]CatalogEntry, 0, len(grouped))

	for _, entry := range grouped {
		slices.Sort(entry.Actions)
		slices.Sort(entry.Grants)

		entries = append(entries, *entry)
	}

	slices.SortFunc(entries, func(first CatalogEntry, second CatalogEntry) int {
		if first.Feature != second.Feature {
			return strings.Compare(first.Feature, second.Feature)
		}

		return strings.Compare(first.Subfeature, second.Subfeature)
	})

	return entries
}

// Not catalog keys, so they cannot be enumerated from it.
func Wildcards() []string {
	return []string{
		grantNamespace + "::*::*",
		grantNamespace + "::*::{ACTION}",
		grantNamespace + "::{feature}::*",
	}
}

// Only ever called on a string that already matched one of the two patterns.
func splitGrant(grant string) (feature string, subfeature string, action string) {
	segments := strings.Split(grant, grantSeparator)

	if len(segments) == 4 {
		subfeature = segments[2]
	}

	return segments[1], subfeature, segments[len(segments)-1]
}

func unionMethods(current []string, extra []string) []string {
	methods := slices.Clone(current)

	for _, method := range extra {
		if !containsFold(methods, method) {
			methods = append(methods, method)
		}
	}

	return methods
}
