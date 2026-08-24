package models

import "encoding/json"

// Key is absent on purpose: it is the identifier the seeds and the payloads
// resolve against, so it must never be writable.
type ProfileUpdateDao struct {
	Name        *string
	Permissions *json.RawMessage
	Metadata    *json.RawMessage
}
