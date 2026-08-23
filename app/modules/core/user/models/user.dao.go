package models

import "encoding/json"

// UserUpdateDao is the allow list of updatable columns of entity.User.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
type UserUpdateDao struct {
	ProfileId        *string
	Name             *string
	Email            *string
	Phone            *string
	VerifyEmail      *bool
	TwoFactorEnabled *bool
	PasswordHash     *string
	Metadata         *json.RawMessage
}
