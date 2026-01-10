package services

import (
	jm "auth_service/app/modules/jwt"
	sr "auth_service/app/modules/session/repository"
	e "auth_service/common/errors"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"
	"errors"
	"fmt"
	"strings"

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
	// user *entity.User,
	// request dto.RequestInfo,
) (string, error) {
	if app.TokenType == "JWT" {
		texts := strings.Split(token, " ")

		if len(texts) != 2 {
			return "", e.ThrowBadRequest("Authorization token in wrong format")
		}

		bearer := texts[0]
		token = texts[1]

		if bearer != "Bearer" {
			return "", e.ThrowBadRequest("Authorization token in wrong format")
		}

		fmt.Println("JWT: ", token)
		payload, err := this.jwtService.ParseAuthToken(token, app.SecretKey)

		if err != nil {
			fmt.Println(err.Error())
			if errors.Is(err, jwt.ErrTokenExpired) {
				return "", e.ThrowUnauthorizedError("Token is expired!")
			}
			if errors.Is(err, jwt.ErrSignatureInvalid) || errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				return "", e.ThrowUnauthorizedError("Token is invalid!")
			}
			return "", e.ThrowBadRequest("Authorization malformatted")
		}

		utils.PrintObj(payload)

		session, err := this.sessionRepository.FindWhere(
			entity.Session{
				ID:          payload.SessionId,
				Invalidated: false,
			},
			"User",
		)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", e.ThrowUnauthorizedError("Session doesnt exist")
			}
			return "", e.ThrowInternalServerError("Unable to find session")
		}

		if session.Invalidated {
			return "", e.ThrowUnauthorizedError("Session was invalidated, Create a new one!")
		}

		if session.Token != payload.Token {
			return "", e.ThrowUnauthorizedError("Incorrect token!")
		}

		utils.PrintObj(session)
	}

	return "", nil
}
