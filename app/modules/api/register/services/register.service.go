package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"auth_service/app/models/dto"
	os "auth_service/app/modules/core/otp/services"
	"auth_service/app/modules/core/profile/services"
	ur "auth_service/app/modules/core/user/repository"
	upr "auth_service/app/modules/core/user_pool/repository"
	hs "auth_service/app/modules/utils/hash/services"

	"auth_service/common/constants"
	e "auth_service/common/errors"
	entity "auth_service/infra/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ IRegisterService = &RegisterService{}

type RegisterService struct {
	userPoolRepository upr.IUserPoolRepository
	userRepository     ur.IUserRepository
	logger             *zap.Logger
	hashService        hs.IHashService
	profileService     services.IProfileService
	otpService         os.IOtpService
}

func NewRegisterService(userPoolRepository upr.IUserPoolRepository, userRepository ur.IUserRepository, profileService services.IProfileService, logger *zap.Logger, hashService hs.IHashService, otpService os.IOtpService) *RegisterService {

	return &RegisterService{
		userPoolRepository: userPoolRepository,
		userRepository:     userRepository,
		profileService:     profileService,
		logger:             logger,
		hashService:        hashService,
		otpService:         otpService,
	}
}

func (this *RegisterService) IsUserAlreadyRegistered(app *entity.App, email string) error {
	_, err := this.userRepository.FindWhere(entity.User{
		Email:       strings.ToLower(email),
		UsersPoolId: app.UsersPool.ID,
	})

	if err == nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return e.ThrowBadRequest("User already exists")
	}

	return nil
}

func (this *RegisterService) RegisterWithPassword(app *entity.App, userData dto.RegisterPayloadWithPassoword) (*entity.User, error) {

	// TODO: move this logic to a permission guard or something like that
	if !slices.Contains(app.LoginTypes, "WITH_PASSWORD") {
		return nil, e.ThrowNotAllowed("This app does not allow login with password")
	}

	err := this.IsUserAlreadyRegistered(app, userData.Email)

	if err != nil {
		return nil, err
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

	// TODO: replace the createdUser from a User.entity to a DTO RegisterResponse
	return createdUser, nil
}

func (this *RegisterService) RegisterWithOtp(app *entity.App, userData dto.RegisterPayloadWithOtp) (*entity.User, error) {
	if !slices.Contains(app.LoginTypes, "WITH_OTP") {
		return nil, e.ThrowNotAllowed("This app does not allow login with OTP")
	}

	err := this.IsUserAlreadyRegistered(app, userData.Email)

	if err != nil {
		return nil, err
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

	user, err := this.userRepository.Create(entity.User{
		Email:       userData.Email,
		Name:        userData.Name,
		Phone:       phone,
		UsersPoolId: app.UsersPool.ID,
		ProfileId:   profile.ID,
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

	return createdUser, nil
}

func (this *RegisterService) CompareOtpMetadataWithPayload(otp *entity.Otp, userData dto.RegisterPayloadWithOtp) (*dto.OtpRegisterActionMetadata, error) {
	var metadata dto.OtpRegisterActionMetadata

	if err := json.Unmarshal(otp.Metadata, &metadata); err != nil {
		return nil, e.ThrowInternalServerError("Failed to parse OTP metadata")
	}

	if !strings.EqualFold(metadata.Email, userData.Email) {
		return nil, e.ThrowBadRequest("Email provided does not match the one used to generate the OTP")
	}

	if metadata.Name != userData.Name {
		return nil, e.ThrowBadRequest("Name provided does not match the one used to generate the OTP")
	}

	return &metadata, nil
}

func (service *RegisterService) Register() error {

	return nil
}
