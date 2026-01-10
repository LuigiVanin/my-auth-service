package services

import (
	"auth_service/app/models/dto"
	jm "auth_service/app/modules/jwt"
	sr "auth_service/app/modules/session/repository"
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
}

func NewAuthorizeService(jwtService jm.IJwtService, sessionRepository sr.ISessionRepository) IAuthorizeService {
	return &AuthorizeService{
		jwtService:        jwtService,
		sessionRepository: sessionRepository,
	}
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

	switch app.TokenType {
	case "JWT":
		payload, err := this.jwtService.ParseAuthToken(token, app.SecretKey)

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, e.ThrowTokenExpiredError("Token is expired!")
			}
			if errors.Is(err, jwt.ErrSignatureInvalid) || errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				return nil, e.ThrowUnauthorizedError("Token is invalid!")
			}
			return nil, e.ThrowBadRequest("Authorization malformatted")
		}

		session, err := this.sessionRepository.FindWhere(
			entity.Session{
				ID:          payload.SessionId,
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

		if session.ExpiresAt.Compare(time.Now()) < 0 {
			return nil, e.ThrowTokenExpiredError("Session is expired!")
		}

		if session.Invalidated {
			return nil, e.ThrowUnauthorizedError("Session was invalidated, Create a new one!")
		}

		if session.Token != payload.Token {
			return nil, e.ThrowUnauthorizedError("Incorrect token!")
		}

		if session.IpAddress != ip {
			return nil, e.ThrowUnauthorizedError("IP Address mismatch!")
		}

		return &dto.AuthorizeReponse{
			User:      session.User,
			SessionId: payload.SessionId,
			Appid:     app.ID,
			ExpiresAt: session.ExpiresAt,
			TokenType: app.TokenType,

			Authorized: true,
		}, nil
	case "SESSION_UUID":
		return nil, e.ThrowNotImplementedError("SESSION UUID not implemented yet")
	}

	return nil, e.ThrowInternalServerError("Auth method notidentified")
}
