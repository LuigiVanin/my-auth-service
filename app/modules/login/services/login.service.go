package services

import (
	"auth_service/app/models/dto"
	hs "auth_service/app/modules/hash/services"
	jwt "auth_service/app/modules/jwt"
	ss "auth_service/app/modules/session/services"
	ur "auth_service/app/modules/user/repository"
	"fmt"

	e "auth_service/common/errors"
	entity "auth_service/infra/entities"
	"errors"
	"slices"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ ILoginService = &LoginService{}

type LoginService struct {
	userRepository ur.IUserRepository
	hashService    hs.IHashService
	sessionService ss.ISessionService
	jwtService     jwt.IJwtService
	logger         *zap.Logger
}

func NewLoginService(userRepository ur.IUserRepository, hashService hs.IHashService, jwtService jwt.IJwtService, sessionService ss.ISessionService, logger *zap.Logger) *LoginService {

	return &LoginService{
		userRepository: userRepository,
		hashService:    hashService,
		sessionService: sessionService,
		jwtService:     jwtService,
		logger:         logger,
	}
}

func (this *LoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword, request dto.RequestInfo) (*dto.LoginResponse, error) {
	// TODO: move this logic to a permission guard or something like that
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

	encryptedRefreshToken, err := this.sessionService.EncryptSessionToken(
		session.ID,
		session.RefreshToken,
		app.SecretKey,
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create token")
	}

	if app.TokenType == "SESSION_UUID" {
		encryptedToken, err := this.sessionService.EncryptSessionToken(
			session.ID,
			session.Token,
			app.SecretKey,
		)

		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to create token")
		}

		return &dto.LoginResponse{
			SessionId:        session.ID,
			Token:            encryptedToken,
			RefreshToken:     encryptedRefreshToken,
			ExpiresAt:        session.ExpiresAt,
			RefreshExpiresAt: session.RefreshExpiresAt,

			User: *user,
		}, nil
	}

	if app.TokenType == "JWT" {
		token, err := this.jwtService.CreateAuthToken(
			dto.AuthPayload{
				User: dto.JwtUser{
					Email: user.Email,
					Name:  user.Name,
					Id:    user.ID,
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

		if err != nil {
			this.logger.Error("Error: ", zap.Error(err))

			fmt.Println("RAW ERROR: ", err.Error())
			return nil, e.ThrowInternalServerError("Failed to create token")
		}

		return &dto.LoginResponse{
			SessionId:        session.ID,
			Token:            token,
			RefreshToken:     encryptedRefreshToken,
			ExpiresAt:        session.ExpiresAt,
			RefreshExpiresAt: session.RefreshExpiresAt,

			User: *user,
		}, nil
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
