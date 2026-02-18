package services

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"auth_service/app/models/dto"
	as "auth_service/app/modules/authorize/services"
	os "auth_service/app/modules/core/otp/services"
	"auth_service/app/modules/core/profile/services"
	ss "auth_service/app/modules/core/session/services"
	ur "auth_service/app/modules/core/user/repository"
	us "auth_service/app/modules/core/user/services"
	upr "auth_service/app/modules/core/user_pool/repository"
	hs "auth_service/app/modules/utils/hash/services"

	e "auth_service/app/errors"
	"auth_service/common/constants"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var _ IRegisterService = &RegisterService{}

type RegisterService struct {
	userPoolRepository upr.IUserPoolRepository
	userRepository     ur.IUserRepository
	userService        us.IUserService
	logger             *zap.Logger
	hashService        hs.IHashService
	profileService     services.IProfileService
	otpService         os.IOtpService
	sessionService     ss.ISessionService
	authorizeService   as.IAuthorizeService
}

func NewRegisterService(
	userPoolRepository upr.IUserPoolRepository,
	userRepository ur.IUserRepository,
	userService us.IUserService,
	profileService services.IProfileService,
	logger *zap.Logger,
	hashService hs.IHashService,
	otpService os.IOtpService,
	sessionService ss.ISessionService,
	authorizeService as.IAuthorizeService,
) *RegisterService {

	return &RegisterService{
		userPoolRepository: userPoolRepository,
		userRepository:     userRepository,
		userService:        userService,
		profileService:     profileService,
		logger:             logger,
		hashService:        hashService,
		otpService:         otpService,
		sessionService:     sessionService,
		authorizeService:   authorizeService,
	}
}

func (this *RegisterService) RegisterWithPassword(app *entity.App, userData dto.RegisterPayloadWithPassoword, request dto.RequestInfo) (*dto.RegisterResponse, error) {

	// TODO: move this logic to a permission guard or something like that
	if !slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	if app.VerifyEmail {
		return nil, e.ThrowNotAllowed("This app requires email verification, to do that you will need to register with OTP!")
	}

	// Check if user already exists
	exists, err := this.userService.IsAlreadyCreated(userData.Email, app)
	if err != nil {
		this.logger.Error("Failed to check if user exists", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to check user existence")
	}
	if exists {
		return nil, e.ThrowBadRequest("User already exists")
	}

	// NOTE: Here I am using a random uuid as Salt. The Salt is stored inside the hashed password in argon2
	// 			 the Compare method take this in consideration.
	hashedPassword, err := this.hashService.HashText(
		userData.Password,
		uuid.New().String(),
	)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to hash password")
	}

	profile, err := this.profileService.GetProfileByAppRole(app.Role)

	if err != nil || profile == nil {
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	if profile.ID == "" {
		return nil, e.ThrowBadRequest(fmt.Sprintf("No profile found for this Role and App. (ROLE: %s)", app.Role))
	}

	var phone string

	if userData.Phone != nil {
		phone = *userData.Phone
	}

	user, err := this.userRepository.Create(entity.User{
		Email:        strings.ToLower(userData.Email),
		PasswordHash: hashedPassword,
		Name:         userData.Name,
		Phone:        phone,
		Metadata:     userData.Metadata,
		UsersPoolId:  app.UsersPool.ID,
		ProfileId:    profile.ID,
	})

	this.logger.Info("User created Successfully!", zap.Uint("userID", user.ID), zap.String("email", user.Email))

	if err != nil {
		this.logger.Error("Failed to create user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create user")
	}

	createdUser, err := this.userRepository.FindWhere(entity.User{
		ID: user.ID,
	}, "Profile")

	if err != nil {
		this.logger.Error("Failed to find user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	// Create session for the newly registered user
	session, err := this.sessionService.CreateNew(app, createdUser, request, "WITH_PASSWORD")
	if err != nil {
		this.logger.Error("Failed to create session", zap.Error(err))
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	// Populate User field in session for CreateAuthorizationCredentials
	session.User = *createdUser

	// Generate authorization credentials
	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)
	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	return &dto.RegisterResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             *createdUser,
	}, nil
}

func (this *RegisterService) RegisterWithOtp(app *entity.App, userData dto.RegisterPayloadWithOtp, request dto.RequestInfo) (*dto.RegisterResponse, error) {
	// if !slices.Contains(app.LoginTypes, "WITH_OTP") {
	// 	return nil, e.ThrowNotAllowed("This app does not allow login with OTP")
	// }

	if userData.Password == "" && slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowUnprocessableEntity("Password is required to register with OTP",
			utils.JSON{
				"errors": utils.JSON{
					"field": "Password",
					"tag":   "password",
					"param": "",
					"value": "-",
				},
			},
		)
	}

	// Check if user already exists
	exists, err := this.userService.IsAlreadyCreated(userData.Email, app)
	if err != nil {
		this.logger.Error("Failed to check if user exists", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to check user existence")
	}
	if exists {
		return nil, e.ThrowBadRequest("User already exists")
	}

	profile, err := this.profileService.GetProfileByAppRole(app.Role)

	if err != nil || profile == nil {
		return nil, e.ThrowInternalServerError("Failed to find profile")
	}

	if profile.ID == "" {
		return nil, e.ThrowBadRequest(fmt.Sprintf("No profile found for this Role and App. (ROLE: %s)", app.Role))
	}

	var phone string

	if userData.Phone != nil {
		phone = *userData.Phone
	}

	otp, err := this.otpService.ValidateConsumable(userData.Otp, app.ID, constants.ActionRegister)

	if err != nil {
		return nil, err
	}

	// Validate that the data sent matches the OTP metadata
	if _, err := this.CompareOtpMetadataWithPayload(otp, userData); err != nil {
		return nil, err
	}

	go this.otpService.Invalidate(otp.ID)

	var hashedPassword string

	if userData.Password != "" && slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		hashedPassword, err = this.hashService.HashText(userData.Password, uuid.New().String())
		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to hash password")
		}
	}

	user, err := this.userRepository.Create(entity.User{
		Email:        userData.Email,
		Name:         userData.Name,
		Phone:        phone,
		UsersPoolId:  app.UsersPool.ID,
		VerifyEmail:  true,
		ProfileId:    profile.ID,
		PasswordHash: hashedPassword,
	})

	this.logger.Info("User created Successfully!", zap.Uint("userID", user.ID), zap.String("email", user.Email))

	if err != nil {
		this.logger.Error("Failed to create user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to create user")
	}

	createdUser, err := this.userRepository.FindWhere(entity.User{
		ID: user.ID,
	}, "Profile")

	if err != nil {
		this.logger.Error("Failed to find user", zap.Error(err))
		return nil, e.ThrowInternalServerError("Failed to find user")
	}

	// Create session for the newly registered user
	session, err := this.sessionService.CreateNew(app, createdUser, request, "WITH_OTP")
	if err != nil {
		this.logger.Error("Failed to create session", zap.Error(err))
		return nil, err
	}

	this.logger.Info("Session created successfully", zap.String("session_id", session.ID))

	// Populate User field in session for CreateAuthorizationCredentials
	session.User = *createdUser

	// Generate authorization credentials
	credentials, err := this.authorizeService.CreateAuthorizationCredentials(app, session)
	if err != nil {
		this.logger.Error("Failed to create authorization credentials", zap.Error(err))
		return nil, err
	}

	return &dto.RegisterResponse{
		SessionId:        session.ID,
		AccessToken:      credentials.AccessToken,
		RefreshToken:     credentials.RefreshToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             *createdUser,
	}, nil
}

func (this *RegisterService) CompareOtpMetadataWithPayload(otp *entity.Otp, userData dto.RegisterPayloadWithOtp) (*dto.OtpRegisterActionMetadata, error) {
	var metadata dto.OtpRegisterActionMetadata

	if err := json.Unmarshal(otp.Metadata, &metadata); err != nil {
		return nil, e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if !strings.EqualFold(metadata.Payload.Email, userData.Email) {
		return nil, e.ThrowBadRequest("Email provided does not match the one used to generate the OTP")
	}

	if metadata.Payload.Name != userData.Name {
		return nil, e.ThrowBadRequest("Name provided does not match the one used to generate the OTP")
	}

	return &metadata, nil
}

func (service *RegisterService) Register() error {

	return nil
}
