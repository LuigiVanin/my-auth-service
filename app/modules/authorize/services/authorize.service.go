package services

import (
	"auth_service/app/models/dto"
	jm "auth_service/app/modules/jwt"
	sr "auth_service/app/modules/session/repository"
	ss "auth_service/app/modules/session/services"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var _ IAuthorizeService = &AuthorizeService{}

type AuthorizeService struct {
	jwtService        jm.IJwtService
	sessionRepository sr.ISessionRepository
	sessionService    ss.ISessionService
}

func NewAuthorizeService(jwtService jm.IJwtService, sessionRepository sr.ISessionRepository, sessionService ss.ISessionService) IAuthorizeService {
	return &AuthorizeService{
		jwtService:        jwtService,
		sessionRepository: sessionRepository,
		sessionService:    sessionService,
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

	return &dto.AuthorizeReponse{
		User:      session.User,
		SessionId: session.ID,
		Appid:     app.ID,
		ExpiresAt: session.ExpiresAt,
		TokenType: app.TokenType,

		Authorized: true,
	}, nil

}
