package authorize_test

import (
	"testing"
	"time"

	"auth_service/app/modules/authorize/services"
	ps "auth_service/app/modules/core/participant/services"
	entity "auth_service/infra/entities"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	sessionId    = "6c2d1c11-8ad5-4dc4-a5ee-79ae2b9f3270"
	refreshToken = "refresh-token"
	callerIp     = "::1"
)

type refreshMocks struct {
	sessionRepo        *mock.MockSessionRepository
	sessionService     *mock.MockSessionService
	participantService *mock.MockParticipantService
}

// supersededSession is the row the service meets after another login rotated the
// user onto a newer session: still stored, still inside its refresh window, and
// carrying the very token the caller is presenting.
func supersededSession() *entity.Session {
	return &entity.Session{
		ID:               sessionId,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IpAddress:        callerIp,
		Invalidated:      true,
		LoginType:        "WITH_PASSWORD",
		User:             entity.User{ID: 1, CurrentOrganization: &entity.Organization{ID: "org"}},
	}
}

func newRefreshService(stored *entity.Session) (services.IAuthorizeService, *refreshMocks) {
	mocks := &refreshMocks{
		sessionRepo:        new(mock.MockSessionRepository),
		sessionService:     new(mock.MockSessionService),
		participantService: new(mock.MockParticipantService),
	}

	mocks.sessionService.
		On("DecryptSessionToken", testifymock.Anything, testifymock.Anything).
		Return(sessionId, refreshToken, nil)

	mocks.sessionRepo.
		On("FindOne", testifymock.Anything, testifymock.Anything).
		Return(stored, nil)

	rotated := &entity.Session{
		ID:               "new-session",
		Token:            "new-token",
		RefreshToken:     "new-refresh-token",
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		User:             entity.User{ID: 1},
	}

	mocks.sessionService.
		On("CreateNew", testifymock.Anything, testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return(rotated, nil)
	mocks.sessionService.
		On("EncryptSessionToken", testifymock.Anything, testifymock.Anything, testifymock.Anything).
		Return("encrypted", nil)

	mocks.participantService.
		On("FindForCurrentOrganization", testifymock.Anything).
		Return(&ps.ResolvedParticipation{
			Participant: &entity.Participant{Profile: &entity.Profile{}},
		}, nil)

	service := services.NewAuthorizeService(
		nil, mocks.sessionRepo, mocks.sessionService, nil, nil, nil, mocks.participantService, zap.NewNop(),
	)

	return service, mocks
}

// A session invalidated by a newer one is exactly what the refresh token exists
// to replace: refusing it logs the caller out because some other client signed in.
func TestRefreshAcceptsASupersededSession(t *testing.T) {
	service, _ := newRefreshService(supersededSession())

	res, err := service.Refresh(&entity.App{ID: "app", TokenType: "SESSION_UUID"}, "Bearer "+refreshToken, callerIp)

	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestRefreshStillRefusesAnExpiredRefreshWindow(t *testing.T) {
	stored := supersededSession()
	stored.RefreshExpiresAt = time.Now().Add(-time.Hour)

	service, _ := newRefreshService(stored)

	_, err := service.Refresh(&entity.App{ID: "app", TokenType: "SESSION_UUID"}, "Bearer "+refreshToken, callerIp)

	assert.Error(t, err)
}

func TestRefreshStillRefusesAWrongToken(t *testing.T) {
	stored := supersededSession()
	stored.RefreshToken = "another-token"

	service, _ := newRefreshService(stored)

	_, err := service.Refresh(&entity.App{ID: "app", TokenType: "SESSION_UUID"}, "Bearer "+refreshToken, callerIp)

	assert.Error(t, err)
}
