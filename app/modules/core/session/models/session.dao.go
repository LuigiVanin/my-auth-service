package models

import "time"

// SessionUpdateDao is the allow list of updatable columns of entity.Session.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
type SessionUpdateDao struct {
	Invalidated *bool
	LastUsedAt  *time.Time
}
