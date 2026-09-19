package models

import "time"

// SessionUpdateDao is the allow list of updatable columns of entity.Session.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
//
// RefreshExpiresAt is here for the revocation pair in the repository and for
// nothing else - the module has no controller, so no request body reaches this
// struct. Closing the refresh window is how a revoked session is told apart from
// one merely superseded by a newer login; see the comment on Revoke.
type SessionUpdateDao struct {
	Invalidated      *bool
	LastUsedAt       *time.Time
	RefreshExpiresAt *time.Time
}
