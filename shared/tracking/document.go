// Package tracking holds the shape of the `tracking` column and the arithmetic
// that folds one event into it.
//
// It lives in shared, and not in the user or users pool module, because both
// modules maintain a column of this shape and the arithmetic is pure computation
// over two JSON values - the same reason shared/permissions is not a service
// method. Nothing here touches a repository or a database.
//
// See docs/features/2026-09-09-tracking/spec.md.
package tracking

import (
	"encoding/json"
	"slices"
	"time"
)

// Limit caps every collection in the document: periods kept per tag, and login
// events kept per user. The oldest goes out when a new one does not fit.
const Limit = 10

// periodLayout keys a period so that lexicographic order is chronological, which
// is what lets eviction pick the oldest month by sorting the keys.
const periodLayout = "2006-01"

// Tag is the top level key of the column. A tag is the kind of thing being
// tracked, and an entity only accepts the tags it actually maintains.
type Tag string

const (
	TagSignup Tag = "signup"
	TagLogin  Tag = "login"
)

// Document is the whole column. Both halves are pointers so a users pool serializes
// no `login` key and a user serializes no `signup` key.
type Document struct {
	Signup *SignupTracking `json:"signup,omitempty"`
	Login  *LoginTracking  `json:"login,omitempty"`
}

// SignupTracking counts, it never lists: an individual signup is not kept, only
// the month it happened in and the app it came through.
type SignupTracking struct {
	Periods map[string]SignupPeriod `json:"periods"`
}

type SignupPeriod struct {
	Total int            `json:"total"`
	Apps  map[string]int `json:"apps"`
}

// LoginTracking is the one place the "counters, never items" rule is broken on
// purpose - "the last logins of a user" is a list of events by definition. See
// the spec.
type LoginTracking struct {
	Events []LoginEvent `json:"events"`
}

type LoginEvent struct {
	AppId     string    `json:"app_id"`
	At        time.Time `json:"at"`
	Ip        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	LoginType string    `json:"login_type,omitempty"`
}

// Period is the key a moment falls into, exported so a caller can report which
// bucket it just moved.
func Period(at time.Time) string {
	return at.UTC().Format(periodLayout)
}

// RecordSignup adds one signup to the month of `at`, both to the total of the
// month and to the breakdown of the app it came through.
func RecordSignup(document json.RawMessage, appId string, at time.Time) (json.RawMessage, error) {
	parsed, err := Parse(document)

	if err != nil {
		return nil, err
	}

	if parsed.Signup == nil {
		parsed.Signup = &SignupTracking{}
	}

	if parsed.Signup.Periods == nil {
		parsed.Signup.Periods = map[string]SignupPeriod{}
	}

	key := Period(at)
	period := parsed.Signup.Periods[key]

	if period.Apps == nil {
		period.Apps = map[string]int{}
	}

	period.Total++
	period.Apps[appId]++

	parsed.Signup.Periods[key] = period
	parsed.Signup.Periods = keepNewestPeriods(parsed.Signup.Periods)

	return json.Marshal(parsed)
}

// RecordLogin pushes the event onto the front of the queue and drops whatever no
// longer fits.
func RecordLogin(document json.RawMessage, event LoginEvent) (json.RawMessage, error) {
	parsed, err := Parse(document)

	if err != nil {
		return nil, err
	}

	if parsed.Login == nil {
		parsed.Login = &LoginTracking{}
	}

	events := append([]LoginEvent{event}, parsed.Login.Events...)

	if len(events) > Limit {
		events = events[:Limit]
	}

	parsed.Login.Events = events

	return json.Marshal(parsed)
}

// Parse reads the column. An empty or null column is an empty document, not an
// error: the row may predate the migration that added the field.
func Parse(document json.RawMessage) (Document, error) {
	parsed := Document{}

	if len(document) == 0 {
		return parsed, nil
	}

	err := json.Unmarshal(document, &parsed)

	return parsed, err
}

func keepNewestPeriods(periods map[string]SignupPeriod) map[string]SignupPeriod {
	if len(periods) <= Limit {
		return periods
	}

	keys := make([]string, 0, len(periods))

	for key := range periods {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys[:len(keys)-Limit] {
		delete(periods, key)
	}

	return periods
}
