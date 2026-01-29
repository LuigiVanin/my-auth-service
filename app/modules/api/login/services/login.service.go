package services

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"auth_service/app/models/dto"
	as "auth_service/app/modules/api/authorize/services"
	os "auth_service/app/modules/core/otp/services"
	ss "auth_service/app/modules/core/session/services"
	ur "auth_service/app/modules/core/user/repository"
	hs "auth_service/app/modules/utils/hash/services"
	"auth_service/common/constants"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ ILoginService = &LoginService{}

type LoginService struct {
	userRepository   ur.IUserRepository
	hashService      hs.IHashService
	sessionService   ss.ISessionService
	authorizeService as.IAuthorizeService
	otpService       os.IOtpService
	logger           *zap.Logger
}

func NewLoginService(
	userRepository ur.IUserRepository,
	hashService hs.IHashService,
	sessionService ss.ISessionService,
	authorizeService as.IAuthorizeService,
	otpService os.IOtpService,
	logger *zap.Logger,
) *LoginService {

	return &LoginService{
		userRepository:   userRepository,
		hashService:      hashService,
		sessionService:   sessionService,
		authorizeService: authorizeService,
		otpService:       otpService,
		logger:           logger,
	}
}

func (this *LoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword, request dto.RequestInfo) (*dto.LoginResponse, error) {
	// TODO: move this logic to a permission guard or something like that
	if !slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	user, err := this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(userData.Email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err != nil || user == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// NOTE: If the user is not found in the User Pool, it means that the user is not registered in the app
			// TODO: Maybe is not a great ideia to return 404 here, this could be a hint to badactors
			return nil, e.ThrowNotFound("User not found in User Pool") // NOT FOUND
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
	session, err := this.sessionService.CreateNew(app, user, request, "WITH_PASSWORD")
	if err != nil {
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	// Populate User field in session for CreateAuthorizationCredentials
	session.User = *user

	// Generate authorization credentials
	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)
	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	return &dto.LoginResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             *user,
	}, nil
}

func (this *LoginService) LoginWithOtp(app *entity.App, userData dto.LoginPayloadWithOtp, request dto.RequestInfo) (*dto.LoginResponse, error) {
	if !slices.Contains(app.LoginTypes, "WITH_OTP") {
		return nil, e.ThrowNotAllowed("This app does not allow login with OTP")
	}

	user, err := this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(userData.Email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err != nil || user == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.ThrowNotFound("User not found in User Pool")
		}

		return nil, e.ThrowInternalServerError("Failed to find user in User Pool")
	}

	this.logger.Info("User found in User Pool", zap.Any("user", user))

	otp, err := this.otpService.ValidateConsumable(userData.Otp, app.ID, constants.ActionLogin)

	if err != nil {
		return nil, err
	}

	err = this.CompareOtpMetadataWithPayload(otp, userData)

	if err != nil {
		return nil, err
	}

	session, err := this.sessionService.CreateNew(app, user, request, "WITH_PASSWORD")
	if err != nil {
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	session.User = *user

	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)

	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	go this.otpService.Invalidate(otp.ID)

	return &dto.LoginResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             *user,
	}, nil
}

func (this *LoginService) CompareOtpMetadataWithPayload(otp *entity.Otp, userData dto.LoginPayloadWithOtp) error {
	var metadata dto.OtpLoginActionMetadata

	if err := json.Unmarshal(otp.Metadata, &metadata); err != nil {
		return e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if !strings.EqualFold(metadata.Payload.Email, userData.Email) {
		return e.ThrowBadRequest("Email provided does not match the one used to generate the OTP")
	}

	return nil
}
