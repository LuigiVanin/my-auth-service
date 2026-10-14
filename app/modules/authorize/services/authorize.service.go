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
	ur "auth_service/app/modules/core/user/repository"
	us "auth_service/app/modules/core/user/services"
	hs "auth_service/app/modules/utils/hash/services"
	jm "auth_service/app/modules/utils/jwt"
	jdto "auth_service/app/modules/utils/jwt/models"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	sharedDto "auth_service/shared/models"
	repo "auth_service/shared/repository"
	"auth_service/shared/storage"
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
	userRepository     ur.IUserRepository
	userService        us.IUserService
	otpService         os.IOtpService
	hashService        hs.IHashService
	participantService ps.IParticipantService
	txManager          repo.ITransactionManager
	logger             *zap.Logger
	urls               storage.IUrlResolver
}

// The user repository sits next to the user service on purpose: ChangeEmail
// writes the user and the sessions in one transaction, and inside a unit of work
// a write goes through the repository so it lands on the shared repo.Option.
// Every read still goes through the service.
func NewAuthorizeService(
	jwtService jm.IJwtService,
	sessionRepository sr.ISessionRepository,
	sessionService ss.ISessionService,
	userRepository ur.IUserRepository,
	userService us.IUserService,
	otpService os.IOtpService,
	hashService hs.IHashService,
	participantService ps.IParticipantService,
	txManager repo.ITransactionManager,
	logger *zap.Logger,
	urls storage.IUrlResolver,
) IAuthorizeService {
	return &AuthorizeService{
		jwtService:         jwtService,
		sessionRepository:  sessionRepository,
		sessionService:     sessionService,
		userRepository:     userRepository,
		userService:        userService,
		otpService:         otpService,
		hashService:        hashService,
		participantService: participantService,
		txManager:          txManager,
		logger:             logger,
		urls:               urls,
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
			UserView: *sharedDto.NewUserView(&session.User, this.urls),
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
		User:      *sharedDto.NewUserView(&session.User, this.urls),
		SessionId: session.ID,
		Appid:     app.ID,
		ExpiresAt: session.ExpiresAt,
		TokenType: app.TokenType,

		Authorized: true,
	}, nil

}

// Revoke is the logout. It is a single mutation, so it goes through the session
// service rather than through the repository this module also holds.
func (this *AuthorizeService) Revoke(sessionId string) error {
	return this.sessionService.Revoke(sessionId)
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
	verified := true
	now := time.Now()

	_, err := this.userService.Update(
		entity.User{ID: user.ID},
		udto.UserUpdateDao{
			VerifyEmail:       &verified,
			VerifiedEmailDate: &now,
		},
	)

	if err != nil {
		return err
	}

	// NOTE: keeps the in memory user consistent with the row that was just
	// written, the same way SetPassword does - the caller serializes it back.
	user.VerifyEmail = verified
	user.VerifiedEmailDate = &now

	return nil
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

func (this *AuthorizeService) ChangeEmail(app *entity.App, user *entity.User, payload dto.ChangeEmail) (*entity.User, error) {
	otpResponse, err := this.otpService.ValidateConsumable(payload.Otp, app.ID, constants.ActionChangeEmail)

	if err != nil {
		return nil, err
	}

	// The OTP was sent to the address being moved to, so the email it stores is the
	// new one - not the identity of the caller, which comes from the token.
	if _, err := this.checkOtpMetadata(*otpResponse, payload.Email); err != nil {
		return nil, err
	}

	newEmail := strings.ToLower(payload.Email)

	if newEmail == strings.ToLower(user.Email) {
		return nil, e.ThrowBadRequest("The new email is the one already on the account")
	}

	// Checked again, having been checked when the code was generated: the address
	// can be taken in between. The unique index is still the net below this, but it
	// would surface as a 500 instead of naming the conflict.
	taken, err := this.userService.IsAlreadyCreated(newEmail, app)

	if err != nil {
		this.logger.Error("Failed to check the new email", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to check the new email")
	}

	if taken {
		return nil, e.ThrowUserAlreadyExists("Another user of this pool already uses this email")
	}

	if err := this.writeEmailChange(app, user, newEmail); err != nil {
		return nil, err
	}

	this.otpService.Invalidate(otpResponse.ID)

	return user, nil
}

// writeEmailChange is the unit of work: moving the address without closing the
// sessions would leave tokens alive under the identity that no longer exists, so
// the two writes commit together or neither does.
func (this *AuthorizeService) writeEmailChange(app *entity.App, user *entity.User, newEmail string) error {
	tx, err := this.txManager.Tx()

	if err != nil {
		return e.ThrowInternalServerError("Failed to open transaction")
	}

	defer tx.Rollback()

	option := repo.Option{Tx: tx}

	verified := true
	now := time.Now()

	affected, err := this.userRepository.Update(
		entity.User{ID: user.ID},
		udto.UserUpdateDao{
			Email:             &newEmail,
			VerifyEmail:       &verified,
			VerifiedEmailDate: &now,
		},
		option,
	)

	if err != nil {
		this.logger.Error("Failed to write the new email", zap.Error(err))
		return e.ThrowInternalServerError("Failed to change the email")
	}

	if affected == 0 {
		return e.ThrowNotFound("User not found")
	}

	if _, err := this.sessionRepository.RevokeAllForUser(user.ID, app.ID, option); err != nil {
		this.logger.Error("Failed to revoke the sessions of the user", zap.Error(err))
		return e.ThrowInternalServerError("Failed to close the sessions of the user")
	}

	if err := tx.Commit(); err != nil {
		return e.ThrowInternalServerError("Failed to commit transaction")
	}

	// NOTE: keeps the in memory user consistent with the row that was just written,
	// the same way SetPassword does. Reading it back instead would put a query that
	// can fail *after* the commit in front of the response, and answering 500 for a
	// change that already happened is worse than the extra assignment.
	user.Email = newEmail
	user.VerifyEmail = verified
	user.VerifiedEmailDate = &now

	return nil
}

func (this *AuthorizeService) VerifyEmail(app *entity.App, user *entity.User, payload dto.VerifyEmailBody) (*entity.User, error) {
	otpResponse, err := this.otpService.ValidateConsumable(payload.Otp, app.ID, constants.ActionVerifyEmail)

	if err != nil {
		return nil, err
	}

	_, err = this.checkOtpMetadata(*otpResponse, payload.Email)

	if err != nil {
		return nil, err
	}

	stored, err := this.userService.FindUserInPool(payload.Email, app.UsersPool.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	if stored == nil {
		return nil, e.ThrowNotFound("User not found")
	}

	if stored.VerifyEmail {
		return stored, nil
	}

	if err := this.VerifyUserEmail(stored); err != nil {
		return nil, err
	}

	this.otpService.Invalidate(otpResponse.ID)

	return stored, nil
}
