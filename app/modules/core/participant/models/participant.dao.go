package models

import "encoding/json"

// OrganizationId and UserId are absent on purpose: moving a participation is not
// an update, it is a different row.
type ParticipantUpdateDao struct {
	ProfileId *string
	Metadata  *json.RawMessage
}
