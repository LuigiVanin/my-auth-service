package services

import (
	"auth_service/app/modules/core/otp/repository"
	"auth_service/common/constants"
	"auth_service/common/utils"
	entity "auth_service/infra/entities"
	"encoding/json"
	"time"
)

type OtpService struct {
	otpRepository repository.IOtpRepository
}

func NewOtpService(otpRepository repository.IOtpRepository) *OtpService {
	return &OtpService{
		otpRepository: otpRepository,
	}
}

func (this *OtpService) GenerateConsumable(app *entity.App, action constants.AuthAction, metadata map[string]any) (*entity.Otp, error) {
	// TODO: Implement logic to generate consumable OTP
	// This usually involves:
	// 1. Generating a random code
	// 2. Setting expiration
	// 3. Saving to repository

	jsonMetadata, err := json.Marshal(metadata)

	if err != nil {
		return nil, err
	}

	passwordCode := utils.GenerateRandomDigits(6)

	otp, err := this.otpRepository.Create(&entity.Otp{
		UserId:   nil, // OTP generated without a known user
		Action:   string(action),
		Metadata: jsonMetadata,
		AppId:    app.ID,

		Code: passwordCode,

		ExpiresAt: time.Now().Add(time.Minute * 6),
	})

	if err != nil {
		return nil, err
	}

	return otp, nil
}
