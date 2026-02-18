package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	e "auth_service/app/errors"
	"auth_service/app/models/dto"
	os "auth_service/app/modules/core/otp/services"
	sr "auth_service/app/modules/core/session/repository"
	ss "auth_service/app/modules/core/session/services"
	us "auth_service/app/modules/core/user/services"
	hs "auth_service/app/modules/utils/hash/services"
	jm "auth_service/app/modules/utils/jwt"
	"auth_service/common/constants"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ IAuthorizeService = &AuthorizeService{}

type AuthorizeService struct {
	jwtService        jm.IJwtService
	sessionRepository sr.ISessionRepository
	sessionService    ss.ISessionService
	userService       us.IUserService
	otpService        os.IOtpService
	hashService       hs.IHashService
	logger            *zap.Logger
}

func NewAuthorizeService(
	jwtService jm.IJwtService,
	sessionRepository sr.ISessionRepository,
	sessionService ss.ISessionService,
	userService us.IUserService,
	otpService os.IOtpService,
	hashService hs.IHashService,
	logger *zap.Logger,
) IAuthorizeService {
	return &AuthorizeService{
		jwtService:        jwtService,
		sessionRepository: sessionRepository,
		sessionService:    sessionService,
		userService:       userService,
		otpService:        otpService,
		hashService:       hashService,
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
		"User.Profile",
	)

	utils.PrintObj(session)

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
		"User.Profile",
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

	credentials, err := this.CreateAuthorizationCredentials(app, newSession)

	if err != nil {
		return nil, err
	}

	return &dto.RefreshResponse{
		SessionId:        newSession.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        newSession.ExpiresAt,
		RefreshExpiresAt: newSession.RefreshExpiresAt,
		User:             session.User,
	}, nil
}

func (this *AuthorizeService) CreateAuthorizationCredentials(app *entity.App, session *entity.Session) (*AuthorizationCredentials, error) {
	encryptedRefreshToken, err := this.sessionService.EncryptSessionToken(
		session.ID,
		session.RefreshToken,
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
				SessionId:  session.ID,
				Token:      session.Token,
				Time:       session.CreatedAt,
				ExpireTime: uint(app.TokenExpirationTime),
			},
			app.SecretKey,
		)
	case "SESSION_UUID":
		accessToken, err = this.sessionService.EncryptSessionToken(
			session.ID,
			session.Token,
			app.SecretKey,
		)
	default:
		return nil, e.ThrowBadRequest("Invalid token type")
	}

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to generate access token")
	}

	return &AuthorizationCredentials{
		AccessToken:  accessToken,
		RefreshToken: encryptedRefreshToken,
	}, nil
}

func (this *AuthorizeService) Authorize(
	app *entity.App,
	token string,
	ip string,
) (*dto.AuthorizeResponse, error) {
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

	return &dto.AuthorizeResponse{
		User:      session.User,
		SessionId: session.ID,
		Appid:     app.ID,
		ExpiresAt: session.ExpiresAt,
		TokenType: app.TokenType,

		Authorized: true,
	}, nil

}

func (this *AuthorizeService) SetPassword(user *entity.User, newPassword string) error {
	hashedPassword, err := this.hashService.HashText(newPassword, uuid.New().String())
	if err != nil {
		return e.ThrowInternalServerError("Failed to hash password")
	}

	user.PasswordHash = hashedPassword

	if err := this.userService.Update(user); err != nil {
		return e.ThrowInternalServerError("Failed to update user password")
	}

	return nil
}

func (this *AuthorizeService) ResetPassword(app *entity.App, payload dto.ResetPasswordPayload) (*entity.User, error) {

	otpResponse, err := this.otpService.ValidateConsumable(payload.Otp, app.ID, constants.ActionForgotPassword)
	if err != nil {
		return nil, err
	}

	var metadata dto.OtpStoredMetadata

	if err := json.Unmarshal(otpResponse.Metadata, &metadata); err != nil {
		return nil, e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if metadata.Payload.Email == "" {
		return nil, e.ThrowBadRequest("Email is required in OTP metadata")
	}

	if metadata.Payload.Email != payload.Email {
		return nil, e.ThrowUnauthorizedError("Email does not match OTP metadata")
	}

	user, err := this.userService.FindUserInPool(payload.Email, app.UsersPool.ID)

	if err != nil {
		return nil, e.ThrowNotFound("User not found")
	}

	if err := this.SetPassword(user, payload.NewPassword); err != nil {
		return nil, err
	}

	return user, nil
}
