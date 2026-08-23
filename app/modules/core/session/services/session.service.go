package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	e "auth_service/app/errors"
	sr "auth_service/app/modules/core/session/repository"
	cipher "auth_service/app/modules/utils/cipher"
	entity "auth_service/infra/entities"
	dto "auth_service/shared/models"

	"go.uber.org/zap"
)

var _ ISessionService = &SessionService{}

type SessionService struct {
	repository    sr.ISessionRepository
	cipherService cipher.ICipherService
	logger        *zap.Logger
}

func NewSessionService(repository sr.ISessionRepository, cipherService cipher.ICipherService, logger *zap.Logger) *SessionService {
	return &SessionService{
		repository:    repository,
		cipherService: cipherService,
		logger:        logger,
	}
}

func (this *SessionService) CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error) {
	metadata, err := json.Marshal(request)
	if err != nil {
		metadata = []byte("{}")
	}
	metadataRaw := json.RawMessage(metadata)

	session, err := this.repository.Create(entity.Session{
		UserId:           user.ID,
		AppId:            app.ID,
		LoginType:        loginType,
		IpAddress:        request.IpAddress,
		UserAgent:        request.UserAgent,
		ExpiresAt:        time.Now().Add(time.Duration(app.TokenExpirationTime) * time.Second),
		RefreshExpiresAt: time.Now().Add(time.Duration(app.RefreshTokenExpirationTime) * time.Second),
		Metadata:         metadataRaw,
	})

	if err != nil || session.ID == "" {
		return nil, e.ThrowInternalServerError("Failed to create session")
	}

	session, err = this.repository.FindOne(entity.Session{
		ID: session.ID,
	})

	if err != nil || session == nil {
		return nil, e.ThrowInternalServerError("Failed to find the new session")
	}

	go func() {
		_, err := this.repository.InvalidateAllExcept(user.ID, app.ID, session.ID)

		if err != nil {
			this.logger.Error(
				"Failed to invalidate all sessions",
				zap.Error(err),
				zap.Uint("user_id", user.ID),
				zap.String("app_id", app.ID),
				zap.String("session_id", session.ID),
			)
		}
	}()

	return session, nil
}

func (this *SessionService) EncryptSessionToken(sessionId string, token string, secretKey string) (string, error) {
	rawData := fmt.Sprintf("%s|%s", sessionId, token)

	return this.cipherService.EncryptTextIntoToken(
		rawData,
		cipher.CipherOptions{
			OverrideKey: secretKey,
		},
	)
}

func (this *SessionService) DecryptSessionToken(tokenString string, secretKey string) (string, string, error) {
	plaintext, err := this.cipherService.DecryptTokenIntoText(
		tokenString,
		cipher.CipherOptions{
			OverrideKey: secretKey,
		},
	)

	if err != nil {
		return "", "", err
	}

	parts := strings.Split(plaintext, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid token format")
	}

	return parts[0], parts[1], nil
}

func (this *SessionService) UseSession(sessionId string) error {
	_, err := this.repository.TouchLastUsedAt(sessionId)

	return err
}
