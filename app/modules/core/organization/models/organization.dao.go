package models

import "encoding/json"

// UsersPoolId is absent: moving an organization between pools would break the
// invariant that it and its participants share one.
type OrganizationUpdateDao struct {
	Name        *string
	Description *string
	ProfileId   *string
	OwnerUserId *uint
	Metadata    *json.RawMessage
}
