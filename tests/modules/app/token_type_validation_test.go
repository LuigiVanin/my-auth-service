package app_test

import (
	"testing"

	middleware "auth_service/app/middlewares"
	dto "auth_service/app/modules/core/app/models"

	"github.com/stretchr/testify/assert"
)

func validCreatePayload(tokenType string) dto.CreateAppPayload {
	return dto.CreateAppPayload{
		Name:                       "client",
		LoginTypes:                 []string{"WITH_PASSWORD"},
		TokenType:                  tokenType,
		TokenExpirationTime:        900,
		RefreshTokenExpirationTime: 86400,
		VerifyEmail:                true,
		UserPool:                   dto.CreateAppPayloadUserPool{Name: "pool"},
	}
}

func TestCreateAppPayload_AcceptsOnlyJwt(t *testing.T) {
	assert.NoError(t, middleware.Validate(validCreatePayload("JWT")))

	for _, legacy := range []string{"FAST_JWT", "SESSION_UUID"} {
		assert.Error(t, middleware.Validate(validCreatePayload(legacy)), legacy)
	}
}

func TestUpdateApp_RefusesLegacyTokenTypes(t *testing.T) {
	jwt := "JWT"
	assert.NoError(t, middleware.Validate(dto.UpdateApp{TokenType: &jwt}))
	assert.NoError(t, middleware.Validate(dto.UpdateApp{}))

	for _, legacy := range []string{"FAST_JWT", "SESSION_UUID"} {
		value := legacy
		assert.Error(t, middleware.Validate(dto.UpdateApp{TokenType: &value}), legacy)
	}
}
