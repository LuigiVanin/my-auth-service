package app_test

import (
	"testing"

	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/core/app/models"

	"github.com/stretchr/testify/assert"
)

func TestGetAppsQuery_AcceptsKnownFilters(t *testing.T) {
	assert.NoError(t, middleware.Validate(dto.GetAppsQuery{}))
	for _, sort := range []string{"updated", "name", "created"} {
		assert.NoError(t, middleware.Validate(dto.GetAppsQuery{Sort: sort}), sort)
	}
	for _, method := range []string{"WITH_PASSWORD", "WITH_OTP", "WITH_LOGIN"} {
		assert.NoError(t, middleware.Validate(dto.GetAppsQuery{LoginType: method}), method)
	}
	assert.NoError(t, middleware.Validate(dto.GetAppsQuery{Private: "false"}))
}

func TestGetAppsQuery_RefusesUnknownValues(t *testing.T) {
	cases := map[string]dto.GetAppsQuery{
		"sort by signups": {Sort: "signups"},
		"sort by column":  {Sort: "updated_at"},
		"login_type":      {LoginType: "with_password"},
		"private":         {Private: "yes"},
	}
	for name, query := range cases {
		assert.Error(t, middleware.Validate(query), name)
	}
}
