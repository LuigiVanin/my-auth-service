package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/repository"
	us "auth_service/app/modules/core/user/services"
	hs "auth_service/app/modules/utils/hash/services"
	"auth_service/common/constants"
	e "auth_service/common/errors"
	"auth_service/common/utils"
	"auth_service/infra/config"
	entity "auth_service/infra/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ IOtpService = &OtpService{}

const (
	OtpExpirationDuration = 5 * time.Minute
	OtpRateLimitDuration  = time.Minute / 3
	OtpLength             = 6
)

type OtpService struct {
	otpRepository repository.IOtpRepository
	hashService   hs.IHashService
	userService   us.IUserService
	logger        *zap.Logger
	cfg           *config.Config
}

func NewOtpService(otpRepository repository.IOtpRepository, hashService hs.IHashService, userService us.IUserService, logger *zap.Logger, cfg *config.Config) *OtpService {
	return &OtpService{
		otpRepository: otpRepository,
		hashService:   hashService,
		userService:   userService,
		logger:        logger,
		cfg:           cfg,
	}
}

// TODO: it will be necessary to build an rate limit for ip addresses for this function
// TODO: Not allow ather fields to be stored inside of the metada
func (this *OtpService) GenerateConsumable(app *entity.App, payload dto.ConsumableOtpPayload, ip string) (*dto.GenerateConsumableOtpResponse, error) {

	data, _ := utils.MapToStruct[dto.OtpStoredMetadataPayload](payload.Payload)

	storedMetadata := dto.OtpStoredMetadata{
		Ip:      ip,
		Payload: data,
	}

	utils.PrintObj(storedMetadata)

	// Add verification_count to payload if not present
	storedMetadata.VerificationCount = 0

	jsonMetadata, err := json.Marshal(storedMetadata)
	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to marshal metadata")
	}

	// Extract contact based on action
	switch payload.Action {
	case constants.ActionRegister:
		if payload.Contact == "" {
			// Try to get email from payload
			if email, ok := payload.Payload["email"].(string); ok {
				payload.Contact = email
			} else {
				return nil, e.ThrowBadRequest("Email is required for REGISTER action")
			}
		}

		// Check if user already exists for REGISTER action
		exists, err := this.userService.IsAlreadyCreated(payload.Contact, app)
		if err != nil {
			this.logger.Error("Failed to check if user exists", zap.Error(err))
			return nil, e.ThrowInternalServerError("Failed to check user existence")
		}
		if exists {
			return nil, e.ThrowBadRequest("User already exists")
		}
	case constants.ActionLogin:
		if payload.Contact == "" {
			// Try to get email from payload
			if email, ok := payload.Payload["email"].(string); ok {
				payload.Contact = email
			} else {
				return nil, e.ThrowBadRequest("Email is required for LOGIN action")
			}

			exists, err := this.userService.IsAlreadyCreated(payload.Contact, app)
			if err != nil {
				this.logger.Error("Failed to check if user exists", zap.Error(err))
				return nil, e.ThrowInternalServerError("Failed to check user existence")
			}
			if !exists {
				return nil, e.ThrowNotFound("User not found in User Pool")
			}
		}
	default:
		return nil, e.ThrowUnprocessableEntity("Invalid OTP action")
	}

	// Check if there was a recent OTP request for this contact, app, and action
	lastOtp, err := this.otpRepository.FindLastOneWhere(entity.Otp{
		Contact: payload.Contact,
		AppId:   app.ID,
		Action:  string(payload.Action),
	})

	if err == nil && lastOtp != nil {
		// Check if the last OTP was created within the last minute
		timeSinceLastOtp := time.Since(lastOtp.CreatedAt)

		if timeSinceLastOtp < OtpRateLimitDuration {
			remainingTime := OtpRateLimitDuration - timeSinceLastOtp

			return nil, e.ThrowTooManyRequests(
				fmt.Sprintf("Please wait %d seconds before requesting another OTP", int(remainingTime.Seconds())),
			)
		}
	}

	passwordCode := utils.GenerateRandomDigits(OtpLength)

	this.logger.Info(fmt.Sprintf("Generated password code: %s", passwordCode))

	hashedPasswordCode := passwordCode

	this.logger.Info(fmt.Sprintf("Env: %s", this.cfg.App.Env))

	if this.cfg.App.Env != "development" {
		this.logger.Info("Hashing password code")
		hashedPasswordCode, err = this.hashService.HashText(passwordCode, uuid.New().String())

		if err != nil {
			return nil, err
		}
	}

	otp, err := this.otpRepository.Create(&entity.Otp{
		UserId:   nil,
		Action:   string(payload.Action),
		Metadata: jsonMetadata,
		AppId:    app.ID,

		Contact: payload.Contact,
		Code:    hashedPasswordCode,

		ExpiresAt: time.Now().Add(OtpExpirationDuration),
	})

	if err != nil {
		return nil, err
	}

	return &dto.GenerateConsumableOtpResponse{
		OtpId: otp.ID,
		AppId: app.ID,

		Action: payload.Action,

		ExpiresAt: otp.ExpiresAt,
		CreatedAt: otp.CreatedAt,

		Payload: payload.Payload,
	}, nil
}

func (this *OtpService) ValidateConsumable(payload dto.PayloadOtpData, appId string, action constants.AuthAction) (*entity.Otp, error) {
	otp, err := this.otpRepository.FindById(payload.Id)

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {

			return nil, e.ThrowNotFound("Otp not found")
		}

		return nil, e.ThrowInternalServerError("Failed to find otp")
	}

	if otp.Action != string(action) {
		return nil, e.ThrowBadRequest("Invalid OTP action")
	}

	if otp.AppId != appId {
		return nil, e.ThrowBadRequest("Invalid OTP app ID")
	}

	if otp.Invalidated {
		return nil, e.ThrowBadRequest("Otp already used")
	}

	if otp.ExpiresAt.Before(time.Now()) {
		return nil, e.ThrowBadRequest("Otp expired")
	}

	if this.cfg.App.Env != "development" {
		valid, err := this.hashService.Compare(payload.Code, otp.Code)

		if err != nil {
			return nil, e.ThrowInternalServerError("Failed to compare otp code")
		}

		if !valid {
			return nil, e.ThrowBadRequest("Invalid otp code")
		}
	} else {
		if otp.Code != payload.Code {
			return nil, e.ThrowBadRequest("Invalid otp code")
		}
	}

	go func() {
	}()

	return otp, nil
}

func (this *OtpService) Invalidate(otpId string) {
	err := this.otpRepository.Invalidate(otpId)

	if err != nil {
		this.logger.Error("Failed to invalidate otp", zap.Error(err), zap.String("otp_id", otpId))
	}

}
