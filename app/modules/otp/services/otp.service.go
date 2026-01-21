package services

import (
	"auth_service/app/modules/otp/repository"
	"auth_service/common/constants"
	entity "auth_service/infra/entities"
)

type OtpService struct {
	otpRepository repository.IOtpRepository
}

func NewOtpService(otpRepository repository.IOtpRepository) *OtpService {
	return &OtpService{
		otpRepository: otpRepository,
	}
}

func (service *OtpService) GenerateConsumable(action constants.AuthAction) (*entity.Otp, error) {
	// TODO: Implement logic to generate consumable OTP
	// This usually involves:
	// 1. Generating a random code
	// 2. Setting expiration
	// 3. Saving to repository
	return nil, nil
}
