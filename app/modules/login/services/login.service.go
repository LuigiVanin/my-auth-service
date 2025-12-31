package services

import (
	"auth_service/app/models/dto"
	hs "auth_service/app/modules/hash/services"
	ss "auth_service/app/modules/session/services"
	ts "auth_service/app/modules/tokenizer/services"
	ur "auth_service/app/modules/user/repository"
	"strconv"

	e "auth_service/common/errors"
	entity "auth_service/infra/entities"
	"errors"
	"slices"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ ILoginService = &LoginService{}

type LoginService struct {
	userRepository   ur.IUserRepository
	hashService      hs.IHashService
	sessionService   ss.ISessionService
	tokenizerService ts.ITokenizerService
	logger           *zap.Logger
}

func NewLoginService(userRepository ur.IUserRepository, hashService hs.IHashService, sessionService ss.ISessionService, tokenizerService ts.ITokenizerService, logger *zap.Logger) *LoginService {

	return &LoginService{
		userRepository:   userRepository,
		hashService:      hashService,
		sessionService:   sessionService,
		tokenizerService: tokenizerService,
		logger:           logger,
	}
}

func (this *LoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword, request dto.RequestInfo) (*dto.LoginResponse, error) {
	if !slices.Contains(app.LoginTypes, "WITH_LOGIN") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	user, err := this.userRepository.FindWhere(entity.User{
		Email:       userData.Email,
		UsersPoolId: app.UsersPool.ID,
	})

	if err != nil || user == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// NOTE: If the user is not found in the User Pool, it means that the user is not registered in the app
			// TODO: Maybe is not a great ideia to return 404 here, this could be a hint to badactors
			return nil, e.ThrowUnauthorizedError("User not found in User Pool") // NOT FOUND?
		}

		return nil, e.ThrowInternalServerError("Failed to find user in User Pool")
	}

	this.logger.Info("User found in User Pool", zap.Any("user", user))

	compare, err := this.hashService.Compare(userData.Password, user.PasswordHash)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to compare password with hash")
	}

	if !compare {
		return nil, e.ThrowUnauthorizedError("Invalid Credentials")
	}

	// NOTE: Here we are creating a new session for the user and invalidating all the other for this user and app
	session, err := this.sessionService.CreateNew(app, user, request, "WITH_LOGIN")
	if err != nil {
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.Any("session", session))

	token, err := this.tokenizerService.CreateToken(session.ID, session.Token, strconv.Itoa(int(user.ID)))

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create token")
	}

	refreshToken, err := this.tokenizerService.CreateToken(session.ID, session.RefreshToken, strconv.Itoa(int(user.ID)))

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create refresh token")
	}

	if app.TokenType == "SESSION_UUID" {
		return &dto.LoginResponse{
			SessionId:        session.ID,
			Token:            token,
			RefreshToken:     refreshToken,
			ExpiresAt:        session.ExpiresAt,
			RefreshExpiresAt: session.RefreshExpiresAt,

			User: *user,
		}, nil
	}

	if app.TokenType == "JWT" {
		return nil, e.ThrowNotImplementedError("JWT Token is not implemented")
	}

	if app.TokenType == "FAST_JWT" {
		return nil, e.ThrowNotImplementedError("FAST_JWT Token is not implemented")
	}

	return nil, e.ThrowBadRequest("Invalid token type, please contact the administrator")
}

func (this *LoginService) LoginWithOtp(app *entity.App, userData dto.LoginPayloadWithOtp, request dto.RequestInfo) (*dto.LoginResponse, error) {
	if !slices.Contains(app.LoginTypes, "WITH_OTP") {
		return nil, e.ThrowNotAllowed("This app does not allow login with OTP")
	}

	return nil, e.ThrowNotImplementedError("LoginWithOtp is not implemented")
}
