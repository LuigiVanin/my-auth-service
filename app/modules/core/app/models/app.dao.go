package models

import (
	"encoding/json"

	"github.com/lib/pq"
)

// AppUpdateDao is the allow list of updatable columns of entity.App.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
//
// UsersPoolId and ParentAppId are absent: the users of an app live in its pool,
// so moving it would leave every one of them behind - the same invariant
// OrganizationUpdateDao protects.
type AppUpdateDao struct {
	OwnerUserId *uint

	PublicKey *string
	SecretKey *string

	Name                       *string
	LoginTypes                 *pq.StringArray
	TokenType                  *string
	TokenExpirationTime        *int64
	RefreshTokenExpirationTime *int64

	Private     *bool
	VerifyEmail *bool
	Enabled2FA  *bool

	Metadata *json.RawMessage
}
