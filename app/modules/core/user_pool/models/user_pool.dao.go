package models

import "encoding/json"

// UserPoolUpdateDao is the allow list of updatable columns of entity.UsersPool.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
//
// UsersCount is absent on purpose: it is a signup counter, so it is only ever
// moved by UserPoolRepository.IncrementUsersCount and never by a payload.
type UserPoolUpdateDao struct {
	Name        *string
	Description *string
	PublicKey   *string

	DefaultProfileId *string

	Metadata *json.RawMessage
}
