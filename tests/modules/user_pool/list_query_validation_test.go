package user_pool_test

import (
	"testing"

	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/core/user_pool/models"

	"github.com/stretchr/testify/assert"
)

func TestUserPoolListQuery_AcceptsKnownFilters(t *testing.T) {
	assert.NoError(t, middleware.Validate(dto.UserPoolListQuery{}))
	assert.NoError(t, middleware.Validate(dto.UserPoolListQuery{Sort: "signups", Ceiling: "broad", Scope: "system"}))
	assert.NoError(t, middleware.Validate(dto.UserPoolListQuery{Sort: "name", Ceiling: "closed", Scope: "organization"}))
}

func TestUserPoolListQuery_RefusesUnknownValues(t *testing.T) {
	cases := map[string]dto.UserPoolListQuery{
		"sort":    {Sort: "users_count"},
		"ceiling": {Ceiling: "unknown"},
		"scope":   {Scope: "global"},
	}
	for name, query := range cases {
		assert.Error(t, middleware.Validate(query), name)
	}
}
