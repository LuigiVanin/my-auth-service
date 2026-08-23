package models

import (
	"encoding/json"

	"github.com/lib/pq"
)

// AppUpdateDao is the allow list of updatable columns of entity.App.
// Every field is a pointer: nil means "do not touch", a filled pointer is
// written as is, including false / 0 / "".
type AppUpdateDao struct {
	UsersPoolId *string
	OwnerUserId *uint
	ParentAppId *string

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
