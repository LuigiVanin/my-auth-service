package authorize_test

import (
	"errors"
	"testing"

	"auth_service/app/modules/authorize/services"
	"auth_service/shared/storage"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func newRevokeService(revokeErr error) (services.IAuthorizeService, *mock.MockSessionService) {
	sessionService := new(mock.MockSessionService)
	sessionService.On("Revoke", sessionId).Return(revokeErr)

	service := services.NewAuthorizeService(
		nil, nil, sessionService, nil, nil, nil, nil, nil, nil, zap.NewNop(),
		storage.NewUrlResolver("https://cdn.test"),
	)

	return service, sessionService
}

func TestRevokeClosesTheSessionOfTheToken(t *testing.T) {
	service, sessionService := newRevokeService(nil)

	assert.NoError(t, service.Revoke(sessionId))
	sessionService.AssertCalled(t, "Revoke", sessionId)
}

func TestRevokeSurfacesAFailureToWrite(t *testing.T) {
	service, _ := newRevokeService(errors.New("write failed"))

	assert.Error(t, service.Revoke(sessionId))
}
