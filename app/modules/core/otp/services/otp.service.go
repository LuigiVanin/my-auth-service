package services

import (
	"auth_service/app/models/dto"
	"auth_service/app/modules/core/otp/repository"
	hs "auth_service/app/modules/utils/hash/services"
	"auth_service/common/constants"
	e "auth_service/common/errors"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

var _ IOtpService = &OtpService{}

const (
	OtpExpirationDuration = 6 * time.Minute
	OtpLength             = 6
)

type OtpService struct {
	otpRepository repository.IOtpRepository
	hashService   hs.IHashService
	logger        *zap.Logger
}

func NewOtpService(otpRepository repository.IOtpRepository, hashService hs.IHashService, logger *zap.Logger) *OtpService {
	return &OtpService{
		otpRepository: otpRepository,
		hashService:   hashService,
		logger:        logger,
	}
}

// TODO: it will be necessary to build an rate limit for ip addresses for this function
func (this *OtpService) GenerateConsumable(app *entity.App, action constants.AuthAction, metadata map[string]any) (*dto.GenerateConsumableOtpResponse, error) {

	var jsonMetadata []byte
	var metadataJson any

	switch action {
	case constants.ActionRegister:
		registerMetadata, err := utils.MapToStruct[dto.OtpRegisterActionMetadata](metadata)
		if err != nil {
			return nil, e.ThrowUnprocessableEntity("Invalid metadata structure for REGISTER action")
		}

		// Safely access Ip from metadata
		ip, ok := metadata["Ip"].(string)
		if !ok {
			return nil, e.ThrowUnprocessableEntity("Ip address is required in metadata")
		}

		registerMetadata.OtpMetadata.VeficationCount = 0
		registerMetadata.OtpMetadata.Ip = ip

		metadataJson = registerMetadata
	default:
		return nil, e.ThrowUnprocessableEntity("Invalid OTP action")
	}

	jsonMetadata, err := json.Marshal(metadataJson)

	if err != nil {
		return nil, err
	}

	passwordCode := utils.GenerateRandomDigits(OtpLength)

	hashedPasswordCode := passwordCode

	// NOTE: disabling secure implementation from DEVELOPMENT
	// hashedPasswordCode, err := this.hashService.HashText(passwordCode, app.SecretKey)

	// if err != nil {
	// 	return nil, err
	// }

	otp, err := this.otpRepository.Create(&entity.Otp{
		UserId:   nil,
		Action:   string(action),
		Metadata: jsonMetadata,
		AppId:    app.ID,

		Code: hashedPasswordCode,

		ExpiresAt: time.Now().Add(OtpExpirationDuration),
	})

	if err != nil {
		return nil, err
	}

	return &dto.GenerateConsumableOtpResponse{
		OtpId: otp.ID,
		AppId: app.ID,

		Action: action,

		ExpiresAt: otp.ExpiresAt,
		CreatedAt: otp.CreatedAt, // Added CreatedAt to response for completeness

		Metadata: metadata,
	}, nil
}
