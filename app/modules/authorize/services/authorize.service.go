package services

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	e "auth_service/app/errors"
	dto "auth_service/app/modules/authorize/models"
	odto "auth_service/app/modules/core/otp/models"
	os "auth_service/app/modules/core/otp/services"
	ps "auth_service/app/modules/core/participant/services"
	sr "auth_service/app/modules/core/session/repository"
	ss "auth_service/app/modules/core/session/services"
	udto "auth_service/app/modules/core/user/models"
	us "auth_service/app/modules/core/user/services"
	hs "auth_service/app/modules/utils/hash/services"
	jm "auth_service/app/modules/utils/jwt"
	jdto "auth_service/app/modules/utils/jwt/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	sharedDto "auth_service/shared/models"
	repo "auth_service/shared/repository"
	"auth_service/shared/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var _ IAuthorizeService = &AuthorizeService{}

type AuthorizeService struct {
	jwtService         jm.IJwtService
	sessionRepository  sr.ISessionRepository
	sessionService     ss.ISessionService
	userService        us.IUserService
	otpService         os.IOtpService
	hashService        hs.IHashService
	participantService ps.IParticipantService
	logger             *zap.Logger
}

func NewAuthorizeService(
	jwtService jm.IJwtService,
	sessionRepository sr.ISessionRepository,
	sessionService ss.ISessionService,
	userService us.IUserService,
	otpService os.IOtpService,
	hashService hs.IHashService,
	participantService ps.IParticipantService,
	logger *zap.Logger,
) IAuthorizeService {
	return &AuthorizeService{
		jwtService:         jwtService,
		sessionRepository:  sessionRepository,
		sessionService:     sessionService,
		userService:        userService,
		otpService:         otpService,
		hashService:        hashService,
		participantService: participantService,
		logger:             logger,
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

	session, err := this.sessionRepository.FindActive(
		sessionId,
		repo.Option{With: []string{"User", "User.CurrentOrganization", "User.CurrentOrganization.Profile"}},
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Unable to find session")
	}

	if session == nil {
		return nil, e.ThrowUnauthorizedError("Session doesnt exist")
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

	// Not FindActive: a refresh token has to work on a session that was superseded
	// by a newer one, which is the whole point of holding it. Validity is decided
	// below, by RefreshExpiresAt and by the token itself.
	session, err := this.sessionRepository.FindOne(
		entity.Session{ID: sessionId},
		repo.Option{With: []string{"User", "User.CurrentOrganization", "User.CurrentOrganization.Profile"}},
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Unable to find session")
	}

	if session == nil {
		return nil, e.ThrowUnauthorizedError("Session doesnt exist")
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

	// An invalidated session is deliberately accepted here. Today the only thing
	// that sets the flag is CreateNew superseding older sessions, so refusing it
	// would log a caller out because someone else - another device, a test - signed
	// in. A session closed on purpose needs its own column before it can be told
	// apart. See docs/features/pendencias.md.

	if session.IpAddress != ip {
		return nil, e.ThrowUnauthorizedError("IP Address mismatch!")
	}

	// Session Rotation: Create a new session
	reqInfo := sharedDto.RequestInfo{
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

	participation, err := this.participantService.FindForCurrentOrganization(&session.User)

	if err != nil {
		return nil, err
	}

	return &dto.RefreshResponse{
		SessionId:        newSession.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        newSession.ExpiresAt,
		RefreshExpiresAt: newSession.RefreshExpiresAt,
		User: udto.UserResponse{
			User: session.User,
			Profile: &udto.ProfileResponse{
				Profile:     *participation.Participant.Profile,
				Permissions: participation.Permissions,
			},
		},
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
			jdto.AuthPayload{
				User: jdto.JwtUser{
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

	// It used to assign to the `err` of the enclosing function, which raced with
	// the handler reading it.
	utils.Detach(this.logger, "mark the session as used", func() error {
		return this.sessionService.UseSession(session.ID)
	})

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

	if _, err := this.userService.Update(
		entity.User{ID: user.ID},
		udto.UserUpdateDao{PasswordHash: &hashedPassword},
	); err != nil {
		return e.ThrowInternalServerError("Failed to update user password")
	}

	// NOTE: keeps the in memory user consistent with the row that was just written.
	user.PasswordHash = hashedPassword

	return nil
}

func (this *AuthorizeService) VerifyUserEmail(user *entity.User) error {

	value := true
	time := time.Now()

	_, err := this.userService.Update(
		entity.User{ID: user.ID},
		udto.UserUpdateDao{
			VerifyEmail:       &value,
			VerifiedEmailDate: &time,
		},
	)

	return err
}

func (this *AuthorizeService) checkOtpMetadata(otp entity.Otp, email string) (odto.OtpStoredMetadata, error) {

	var metadata odto.OtpStoredMetadata

	if err := json.Unmarshal(otp.Metadata, &metadata); err != nil {
		return metadata, e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if metadata.Payload.Email == "" {
		return metadata, e.ThrowBadRequest("Email is required in OTP metadata")
	}

	if !strings.EqualFold(metadata.Payload.Email, email) {
		return metadata, e.ThrowUnauthorizedError("Email does not match OTP metadata")
	}

	return metadata, nil
}

func (this *AuthorizeService) ResetPassword(app *entity.App, payload dto.ResetPasswordPayload) (*entity.User, error) {

	otpResponse, err := this.otpService.ValidateConsumable(payload.Otp, app.ID, constants.ActionForgotPassword)
	if err != nil {
		return nil, err
	}

	_, err = this.checkOtpMetadata(*otpResponse, payload.Email)

	if err != nil {
		return nil, err
	}

	user, err := this.userService.FindUserInPool(payload.Email, app.UsersPool.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	if user == nil {
		return nil, e.ThrowNotFound("User not found")
	}

	if err := this.SetPassword(user, payload.NewPassword); err != nil {
		return nil, err
	}

	this.otpService.Invalidate(otpResponse.ID)

	return user, nil
}

func (this *AuthorizeService) VerifyEmail(app *entity.App, user *entity.User, payload dto.VerifyEmailBody) error {
	otpResponse, err := this.otpService.ValidateConsumable(payload.Otp, app.ID, constants.ActionVerifyEmail)

	if err != nil {
		return err
	}

	_, err = this.checkOtpMetadata(*otpResponse, payload.Email)

	if err != nil {
		return err
	}

	user, err = this.userService.FindUserInPool(payload.Email, app.UsersPool.ID)

	if err != nil {
		return e.ThrowInternalServerError("Failed to find user")
	}

	if user == nil {
		return e.ThrowNotFound("User not found")
	}

	if !user.VerifyEmail {
		return nil
	}

	if err := this.VerifyUserEmail(user); err != nil {
		return err
	}

	return nil
}
