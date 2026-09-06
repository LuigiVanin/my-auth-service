// Package permissions holds the permission document and the algebra that stacks
// one on top of another.
//
// It lives in shared, and not in the profile module, because every layer of the
// application needs it: the entities carry a resolved document, the guards enforce
// one, the services clamp what they write against one. Making it a service method
// would force a dependency for what is pure computation over two JSON values.
//
// See steering/modules/profiles.md.
package permissions

import "encoding/json"

const Wildcard = "*"

// Document is the shape of profiles.permissions. It has two ways of saying the
// same thing:
//
//	{ "grants": ["as::users::READ"] }
//
//	{ "api": { "/core/users": {
//	    "methods": ["GET"],
//	    "query":   { "skip": "^[0-9]+$" } } } }
//
// Grants are the authoring format and are what a profile carries in practice.
// Api is the administration escape hatch, for the granularity no grant expresses
// - a query regex, one specific path - and it wins the whole path over whatever
// a grant expands to. See steering/modules/profiles.md.
//
// An absent or empty query object allows any query string, same as {"*": "*"}.
type Document struct {
	Api    map[string]Rule `json:"api"`
	Grants []string        `json:"grants"`
}

type Rule struct {
	Query   map[string]string `json:"query"`
	Methods []string          `json:"methods"`
}

// Resolved is what Resolve returns: what is left of a document once every layer
// above it has been applied. It is the only permission shape a caller ever sees.
//
// It differs from Document only in Query. Two layers can constrain the same
// parameter with different regexes, and RE2 has no lookahead, so there is no
// single expression meaning "matches both". Picking one side would grant more than
// the other allows, so every pattern is kept and all of them have to match.
//
// Grants is the summary by resource of the same answer: the grants that survived
// every ceiling above. Api is the truth of what the guard does, and where the two
// diverge Api wins - a grant only half covered by a hand written api layer does
// not appear at all.
type Resolved struct {
	Api    map[string]ResolvedRule `json:"api"`
	Grants []string                `json:"grants,omitempty"`
}

type ResolvedRule struct {
	Query   map[string][]string `json:"query,omitempty"`
	Methods []string            `json:"methods"`
}

func Parse(document json.RawMessage) (Document, error) {
	parsed := Document{}

	if len(document) == 0 {
		return parsed, nil
	}

	err := json.Unmarshal(document, &parsed)

	return parsed, err
}

func (this Document) resolved() Resolved {
	resolved := Resolved{Api: expandGrants(this.Grants), Grants: knownGrants(this.Grants)}

	// api replaces the expansion for a path, never merges with it: grants leave the
	// query open, so a union would erase a regex someone wrote by hand.
	for path, rule := range this.Api {
		resolved.Api[path] = ResolvedRule{
			Methods: rule.Methods,
			Query:   singlePatterns(rule.Query),
		}
	}

	return resolved
}

func singlePatterns(query map[string]string) map[string][]string {
	if len(query) == 0 {
		return nil
	}

	patterns := map[string][]string{}

	for key, pattern := range query {
		patterns[key] = []string{pattern}
	}

	return patterns
}
