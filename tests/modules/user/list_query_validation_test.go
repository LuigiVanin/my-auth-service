package user_test

import (
	"testing"

	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/core/user/models"

	"github.com/stretchr/testify/assert"
)

func TestUserListQuery_AcceptsWellFormedFilters(t *testing.T) {
	assert.NoError(t, middleware.Validate(dto.UserListQuery{}))
	assert.NoError(t, middleware.Validate(dto.UserListQuery{
		PoolId:         "7822c6ec-0db0-4038-8b78-342647e24433",
		ProfileId:      "0b3c3c1e-5a52-4d8b-9a55-0e3c9f8a1b2c",
		EmailVerified:  "false",
		TwoFactor:      "true",
		SignedUpAfter:  "2026-11-01T03:00:00.000Z",
		SignedUpBefore: "2026-12-01T00:00:00-03:00",
	}))
}

// A malformed value has to stop at validation: reaching Postgres as a uuid or a
// timestamp it cannot parse would answer 500.
func TestUserListQuery_RefusesMalformedFilters(t *testing.T) {
	cases := map[string]dto.UserListQuery{
		"profile_id":       {ProfileId: "not-a-uuid"},
		"email_verified":   {EmailVerified: "yes"},
		"two_factor":       {TwoFactor: "1"},
		"signed_up_after":  {SignedUpAfter: "2026-11-01"},
		"signed_up_before": {SignedUpBefore: "01/11/2026"},
	}

	for name, query := range cases {
		assert.Error(t, middleware.Validate(query), name)
	}
}
