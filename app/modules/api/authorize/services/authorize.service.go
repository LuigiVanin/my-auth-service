package services

import (
	"errors"
	"strings"
	"time"

	"auth_service/app/models/dto"
	sr "auth_service/app/modules/core/session/repository"
	ss "auth_service/app/modules/core/session/services"
	jm "auth_service/app/modules/utils/jwt"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ IAuthorizeService = &AuthorizeService{}

type AuthorizeService struct {
	jwtService        jm.IJwtService
	sessionRepository sr.ISessionRepository
	sessionService    ss.ISessionService
	logger            *zap.Logger
}

func NewAuthorizeService(jwtService jm.IJwtService, sessionRepository sr.ISessionRepository, sessionService ss.ISessionService, logger *zap.Logger) IAuthorizeService {
	return &AuthorizeService{
		jwtService:        jwtService,
		sessionRepository: sessionRepository,
		sessionService:    sessionService,
		logger:            logger,
	}
}

func (this *AuthorizeService) FindSessionByToken(token string, tokenType string, secretKey string) (*entity.Session, error) {
	var sessionId string
	var sessionToken string

	switch tokenType {
	case "JWT":
		payload, err := this.jwtService.ParseAuthToken(token, secretKey)

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, e.ThrowTokenExpiredError("Token is expired!")
			}
			if errors.Is(err, jwt.ErrSignatureInvalid) || errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				return nil, e.ThrowUnauthorizedError("Token is invalid!")
			}
			return nil, e.ThrowBadRequest("Authorization token malformatted")
		}

		sessionId = payload.SessionId
		sessionToken = payload.Token

	case "SESSION_UUID":
		var err error
		sessionId, sessionToken, err = this.sessionService.DecryptSessionToken(token, secretKey)

		if sessionId == "" || sessionToken == "" || err != nil {
			return nil, e.ThrowBadRequest("Authorization token malformatted")
		}
	}

	session, err := this.sessionRepository.FindWhere(
		entity.Session{
			ID:          sessionId,
			Invalidated: false,
		},
		"User",
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.ThrowUnauthorizedError("Session doesnt exist")
		}
		return nil, e.ThrowInternalServerError("Unable to find session")
	}

	if session.Token != sessionToken {
		return nil, e.ThrowUnauthorizedError("Incorrect session token!")
	}

	return session, err
}

func (this *AuthorizeService) FindSessionByRefreshToken(token string, tokenType string, secretKey string) (*entity.Session, error) {
	sessionId, sessionToken, err := this.sessionService.DecryptSessionToken(token, secretKey)

	if sessionId == "" || sessionToken == "" || err != nil {
		return nil, e.ThrowBadRequest("Authorization token malformatted")
	}

	session, err := this.sessionRepository.FindWhere(
		entity.Session{
			ID:          sessionId,
			Invalidated: false,
		},
		"User",
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.ThrowUnauthorizedError("Session doesnt exist")
		}
		return nil, e.ThrowInternalServerError("Unable to find session")
	}

	if session.RefreshToken != sessionToken {
		return nil, e.ThrowUnauthorizedError("Incorrect refresh token!")
	}

	return session, err
}

func (this *AuthorizeService) Refresh(
	app *entity.App,
	token string,
	ip string,
) (*dto.RefreshResponse, error) {
	parts := strings.Split(token, " ")

	if len(parts) != 2 {
		return nil, e.ThrowBadRequest("Authorization token in wrong format")
	}

	bearer := parts[0]
	token = parts[1]

	if bearer != "Bearer" {
		return nil, e.ThrowBadRequest("Authorization token in wrong format")
	}

	session, err := this.FindSessionByRefreshToken(token, app.TokenType, app.SecretKey)
	if err != nil {
		return nil, err
	}

	if session.RefreshExpiresAt.Compare(time.Now()) < 0 {
		return nil, e.ThrowTokenExpiredError("Refresh session is expired!")
	}

	if session.Invalidated {
		return nil, e.ThrowUnauthorizedError("Session was invalidated, Create a new one!")
	}

	if session.IpAddress != ip {
		return nil, e.ThrowUnauthorizedError("IP Address mismatch!")
	}

	// Session Rotation: Create a new session
	reqInfo := dto.RequestInfo{
		IpAddress: ip,
		UserAgent: session.UserAgent,
	}

	newSession, err := this.sessionService.CreateNew(app, &session.User, reqInfo, session.LoginType)
	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to rotate session")
	}

	encryptedRefreshToken, err := this.sessionService.EncryptSessionToken(
		newSession.ID,
		newSession.RefreshToken,
		app.SecretKey,
	)
	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create refresh token")
	}

	var accessToken string

	switch app.TokenType {
	case "JWT":
		accessToken, err = this.jwtService.CreateAuthToken(
			dto.AuthPayload{
				User: dto.JwtUser{
					Email: session.User.Email,
					Name:  session.User.Name,
					Id:    session.User.ID,
				},
				AppId:      app.ID,
				UserPoolId: app.UsersPoolId,
				SessionId:  newSession.ID,
				Token:      newSession.Token,
				Time:       newSession.CreatedAt,
				ExpireTime: uint(app.TokenExpirationTime),
			},
			app.SecretKey,
		)
	case "SESSION_UUID":
		accessToken, err = this.sessionService.EncryptSessionToken(
			newSession.ID,
			newSession.Token,
			app.SecretKey,
		)
	default:
		return nil, e.ThrowBadRequest("Invalid token type")
	}

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to generate access token")
	}

	return &dto.RefreshResponse{
		SessionId:        newSession.ID,
		AccessToken:      accessToken,
		RefreshToken:     encryptedRefreshToken,
		ExpiresAt:        newSession.ExpiresAt,
		RefreshExpiresAt: newSession.RefreshExpiresAt,
		User:             session.User,
	}, nil
}

func (this *AuthorizeService) Authorize(
	app *entity.App,
	token string,
	ip string,
) (*dto.AuthorizeReponse, error) {
	texts := strings.Split(token, " ")

	if len(texts) != 2 {
		return nil, e.ThrowBadRequest("Authorization token in wrong format")
	}

	bearer := texts[0]
	token = texts[1]

	if bearer != "Bearer" {
		return nil, e.ThrowBadRequest("Authorization token in wrong format")
	}

	session, err := this.FindSessionByToken(token, app.TokenType, app.SecretKey)

	if err != nil {
		return nil, err
	}

	if session.ExpiresAt.Compare(time.Now()) < 0 {
		return nil, e.ThrowTokenExpiredError("Session is expired!")
	}

	if session.Invalidated {
		return nil, e.ThrowUnauthorizedError("Session was invalidated, Create a new one!")
	}

	if session.IpAddress != ip {
		return nil, e.ThrowUnauthorizedError("IP Address mismatch!")
	}

	go func() {
		err = this.sessionService.UseSession(session.ID)
		if err != nil {
			this.logger.Error("Failed to use session", zap.Error(err), zap.String("session_id", session.ID))
			return
		}

		this.logger.Info("Session used", zap.String("session_id", session.ID))
	}()

	return &dto.AuthorizeReponse{
		User:      session.User,
		SessionId: session.ID,
		Appid:     app.ID,
		ExpiresAt: session.ExpiresAt,
		TokenType: app.TokenType,

		Authorized: true,
	}, nil

}
