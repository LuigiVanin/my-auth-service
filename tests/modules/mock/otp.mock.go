package mock

import (
	dto "auth_service/app/modules/core/otp/models"
	orep "auth_service/app/modules/core/otp/repository"
	os "auth_service/app/modules/core/otp/services"
	entity "auth_service/infra/entities"
	"auth_service/shared/constants"
	repo "auth_service/shared/repository"

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

var _ orep.IOtpRepository = &MockOtpRepository{}

func (this *MockOtpRepository) FindOne(where entity.Otp, options ...repo.Option) (*entity.Otp, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpRepository) Create(otp entity.Otp, options ...repo.Option) (*entity.Otp, error) {
	args := this.Called(otp, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}

func (this *MockOtpRepository) Update(where entity.Otp, data dto.OtpUpdateDao, options ...repo.Option) (int64, error) {
	args := this.Called(where, data, options)
	return args.Get(0).(int64), args.Error(1)
}

func (this *MockOtpRepository) FindLast(where entity.Otp, options ...repo.Option) (*entity.Otp, error) {
	args := this.Called(where, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Otp), args.Error(1)
}
