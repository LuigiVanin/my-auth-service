package permissions

import (
	"encoding/json"
	"slices"
	"strings"
)

// Resolve stacks permission documents from the outermost layer to the innermost
// and returns what survives. Arguments are ordered parent first: each one limits
// everything that comes after it, and none of them can widen what came before.
//
//	permissions.Resolve(organization, participant)
//	permissions.Resolve(pool, organization, participant)
//
// This is the single answer to "what may this caller do". It is what the
// PermissionsGuard enforces, what a specific lookup of a user or an organization
// reports, and what a profile has to be clamped to before it is written.
//
// No document means nothing is granted. One document means it is normalised, not
// trusted: it still goes through the same code path.
func Resolve(documents ...json.RawMessage) (*Resolved, error) {
	resolved := Resolved{Api: map[string]ResolvedRule{}}

	for index, raw := range documents {
		document, err := Parse(raw)

		if err != nil {
			return nil, err
		}

		layer := document.resolved()

		if index == 0 {
			resolved = layer
			continue
		}

		resolved = intersect(resolved, layer)
	}

	resolved.Grants = grantsWithin(resolved.Api)

	return &resolved, nil
}

// IsSubsetOf answers whether child exceeds parent anywhere. It is the check behind
// "a layer only grants what it holds": a profile picked as the ceiling of a pool or
// of an organization has to pass it.
//
// Where a caller authors a document rather than picking one, clamp it with Resolve
// instead of refusing it - see docs/steering/modules/profiles.md.
func IsSubsetOf(child json.RawMessage, parent json.RawMessage) (bool, error) {
	childDoc, err := Parse(child)

	if err != nil {
		return false, err
	}

	parentDoc, err := Parse(parent)

	if err != nil {
		return false, err
	}

	// Resolved on both sides, never raw: iterating childDoc.Api would leave the grants
	// of the candidate unchecked, and a profile is written in grants.
	return withinApi(childDoc.resolved().Api, parentDoc.resolved().Api), nil
}

// IsWithin is IsSubsetOf against a ceiling already resolved, which is what a write
// path holds.
//
// It compares against Api and never against Grants: a document written in api
// declares no grant at all, so comparing lists would leave the platform admin unable
// to author anything.
func IsWithin(child json.RawMessage, parent *Resolved) (bool, error) {
	childDoc, err := Parse(child)

	if err != nil {
		return false, err
	}

	if parent == nil {
		return false, nil
	}

	return withinApi(childDoc.resolved().Api, parent.Api), nil
}

// Whether every path of child is granted by parent. This is the question both
// IsSubsetOf and the survival of a grant ask.
func withinApi(child map[string]ResolvedRule, parent map[string]ResolvedRule) bool {
	for path, childRule := range child {
		parentRule, ok := resolvePath(Resolved{Api: parent}, path)

		if !ok {
			return false
		}

		if !methodsWithin(childRule.Methods, parentRule.Methods) {
			return false
		}

		if !queryWithin(childRule.Query, parentRule.Query) {
			return false
		}
	}

	return true
}

func intersect(parent Resolved, child Resolved) Resolved {
	result := Resolved{Api: map[string]ResolvedRule{}}

	for _, path := range candidatePaths(parent, child) {
		parentRule, ok := resolvePath(parent, path)

		if !ok {
			continue
		}

		childRule, ok := resolvePath(child, path)

		if !ok {
			continue
		}

		methods := intersectMethods(parentRule.Methods, childRule.Methods)

		// A path whose methods cancel out grants nothing.
		if len(methods) == 0 {
			continue
		}

		result.Api[path] = ResolvedRule{
			Methods: methods,
			Query:   intersectQuery(parentRule.Query, childRule.Query),
		}
	}

	return result
}

// A concrete key of one side is a candidate even when the other side does not
// declare it, because the other side may cover it through its wildcard. That is
// what makes a participant of "*" collapse onto the ceiling of its organization
// instead of widening it.
func candidatePaths(parent Resolved, child Resolved) []string {
	seen := map[string]bool{}
	paths := []string{}

	for _, document := range []Resolved{parent, child} {
		for path := range document.Api {
			if path == Wildcard || seen[path] {
				continue
			}

			seen[path] = true
			paths = append(paths, path)
		}
	}

	_, parentWild := parent.Api[Wildcard]
	_, childWild := child.Api[Wildcard]

	if parentWild && childWild {
		paths = append(paths, Wildcard)
	}

	slices.Sort(paths)

	return paths
}

// NOTE: exact match, never a regex one, unlike PermissionsGuard. Documents are
// keyed by registered route patterns, so there is nothing to normalise between
// two of them; turning ":id" into a regex only makes sense against a real path.
func resolvePath(document Resolved, path string) (ResolvedRule, bool) {
	if rule, ok := document.Api[path]; ok {
		return rule, true
	}

	if rule, ok := document.Api[Wildcard]; ok {
		return rule, true
	}

	return ResolvedRule{}, false
}

func intersectMethods(parent []string, child []string) []string {
	parentWild := containsFold(parent, Wildcard)
	childWild := containsFold(child, Wildcard)

	switch {
	case parentWild && childWild:
		return []string{Wildcard}
	case parentWild:
		return slices.Clone(child)
	case childWild:
		return slices.Clone(parent)
	}

	methods := []string{}

	for _, method := range parent {
		if containsFold(child, method) && !containsFold(methods, method) {
			methods = append(methods, method)
		}
	}

	return methods
}

// A nil result means any query string is allowed.
func intersectQuery(parent map[string][]string, child map[string][]string) map[string][]string {
	parentOpen := isQueryOpen(parent)
	childOpen := isQueryOpen(child)

	switch {
	case parentOpen && childOpen:
		return nil
	case parentOpen:
		return child
	case childOpen:
		return parent
	}

	query := map[string][]string{}

	for _, key := range candidateQueryKeys(parent, child) {
		parentPatterns, ok := resolveQueryKey(parent, key)

		if !ok {
			continue
		}

		childPatterns, ok := resolveQueryKey(child, key)

		if !ok {
			continue
		}

		query[key] = mergePatterns(parentPatterns, childPatterns)
	}

	return query
}

func candidateQueryKeys(parent map[string][]string, child map[string][]string) []string {
	seen := map[string]bool{}
	keys := []string{}

	for _, document := range []map[string][]string{parent, child} {
		for key := range document {
			if key == Wildcard || seen[key] {
				continue
			}

			seen[key] = true
			keys = append(keys, key)
		}
	}

	if _, parentWild := parent[Wildcard]; parentWild {
		if _, childWild := child[Wildcard]; childWild {
			keys = append(keys, Wildcard)
		}
	}

	slices.Sort(keys)

	return keys
}

func resolveQueryKey(query map[string][]string, key string) ([]string, bool) {
	if patterns, ok := query[key]; ok {
		return patterns, true
	}

	patterns, ok := query[Wildcard]

	return patterns, ok
}

// Every constraint of both sides is kept: see Resolved.
func mergePatterns(parent []string, child []string) []string {
	patterns := []string{}

	for _, side := range [][]string{parent, child} {
		for _, pattern := range side {
			if pattern == Wildcard || slices.Contains(patterns, pattern) {
				continue
			}

			patterns = append(patterns, pattern)
		}
	}

	if len(patterns) == 0 {
		return []string{Wildcard}
	}

	return patterns
}

// An absent or empty query object allows anything, and so does {"*": "*"}.
func isQueryOpen(query map[string][]string) bool {
	if len(query) == 0 {
		return true
	}

	patterns, ok := query[Wildcard]

	return ok && len(patterns) == 1 && patterns[0] == Wildcard
}

func methodsWithin(child []string, parent []string) bool {
	if containsFold(parent, Wildcard) {
		return true
	}

	// Asking for every method while the parent lists a few is an excess, even when
	// the parent list happens to cover every route registered today.
	if containsFold(child, Wildcard) {
		return false
	}

	for _, method := range child {
		if !containsFold(parent, method) {
			return false
		}
	}

	return true
}

func queryWithin(child map[string][]string, parent map[string][]string) bool {
	if isQueryOpen(parent) {
		return true
	}

	// The child would let through parameters the parent refuses.
	if isQueryOpen(child) {
		return false
	}

	for key, childPatterns := range child {
		parentPatterns, ok := resolveQueryKey(parent, key)

		if !ok {
			return false
		}

		for _, childPattern := range childPatterns {
			if slices.Contains(parentPatterns, Wildcard) || slices.Contains(parentPatterns, childPattern) {
				continue
			}

			// Two concrete patterns cannot be compared for containment; deny.
			return false
		}
	}

	return true
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}

	return false
}
