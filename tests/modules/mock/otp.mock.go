package mock

import (
	dto "auth_service/app/modules/core/otp/models"
	os "auth_service/app/modules/core/otp/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"

	"github.com/stretchr/testify/mock"
)

// MockOtpService represents a mock implementation of IOtpService
type MockOtpService struct {
	mock.Mock
}

func (this *MockOtpService) GenerateConsumable(action constants.AuthAction, app *entity.App, payload dto.ConsumableOtpPayload, ip string) (*dto.GenerateConsumableOtpResponse, error) {
	args := this.Called(action, app, payload, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.GenerateConsumableOtpResponse), args.Error(1)
}

func (this *MockOtpService) ValidateConsumable(payload dto.PayloadOtpData, appId string, action constants.AuthAction) (*entity.Otp, error) {
	args := this.Called(payload, appId, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpService) VerifyConsumable(otpId string, code string, appId string) (*os.VerifyConsumableOtpResponse, error) {
	args := this.Called(otpId, code, appId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*os.VerifyConsumableOtpResponse), args.Error(1)
}

func (this *MockOtpService) Invalidate(otpId string) {
	this.Called(otpId)
}

// MockOtpRepository represents a mock implementation of IOtpRepository
type MockOtpRepository struct {
	mock.Mock
}

func (this *MockOtpRepository) Create(otp *entity.Otp) (*entity.Otp, error) {
	args := this.Called(otp)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpRepository) FindById(otpId string) (*entity.Otp, error) {
	args := this.Called(otpId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpRepository) FindLastOneWhere(where entity.Otp, with ...string) (*entity.Otp, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpRepository) Update(otp *entity.Otp) error {
	args := this.Called(otp)
	return args.Error(0)
}

func (this *MockOtpRepository) Invalidate(otpId string) error {
	args := this.Called(otpId)
	return args.Error(0)
}
