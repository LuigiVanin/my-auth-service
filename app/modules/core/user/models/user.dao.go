package models

import (
	"encoding/json"
	"time"
)

// UserUpdateDao is the allow list of updatable columns of entity.User.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
type UserUpdateDao struct {
	CurrentOrganizationId *string

	Name              *string
	Email             *string
	Phone             *string
	VerifyEmail       *bool
	VerifiedEmailDate *time.Time
	TwoFactorEnabled  *bool
	PasswordHash      *string
	Metadata          *json.RawMessage
}
