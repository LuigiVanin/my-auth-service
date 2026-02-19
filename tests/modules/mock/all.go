package mock

import (
	"auth_service/app/models/dto"
	"auth_service/app/modules/authorize/services"
	os "auth_service/app/modules/core/otp/services"
	"auth_service/app/modules/utils/cipher"
	"auth_service/common/constants"
	entity "auth_service/infra/entities"

	"github.com/stretchr/testify/mock"
)

type MockLoginService struct {
	mock.Mock
}

func (this *MockLoginService) LoginWithPassword(app *entity.App, userData dto.LoginPayloadWithPassoword, request dto.RequestInfo) (*dto.LoginResponse, error) {
	args := this.Called(app, userData, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResponse), args.Error(1)
}

func (this *MockLoginService) LoginWithOtp(app *entity.App, userData dto.LoginPayloadWithOtp, request dto.RequestInfo) (*dto.LoginResponse, error) {
	args := this.Called(app, userData, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResponse), args.Error(1)
}

type MockAppRepository struct {
	mock.Mock
}

func (this *MockAppRepository) FindAppbyIdWithPool(id string) (*entity.App, error) {
	args := this.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) Create(app *entity.App) (*entity.App, error) {
	args := this.Called(app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

func (this *MockAppRepository) FindWhere(where entity.App, with ...string) (*entity.App, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.App), args.Error(1)
}

type MockCipherService struct {
	mock.Mock
}

func (this *MockCipherService) EncryptUuidIntoToken(uuid string, options ...cipher.CipherOptions) (string, error) {
	args := this.Called(uuid, options)
	return args.String(0), args.Error(1)
}

func (this *MockCipherService) DecryptUuidToken(token string, options ...cipher.CipherOptions) (string, error) {
	args := this.Called(token, options)
	return args.String(0), args.Error(1)
}

func (this *MockCipherService) DecryptTokenIntoText(token string, options ...cipher.CipherOptions) (string, error) {
	args := this.Called(token, options)
	return args.String(0), args.Error(1)
}

func (this *MockCipherService) EncryptTextIntoToken(text string, options ...cipher.CipherOptions) (string, error) {
	args := this.Called(text, options)
	return args.String(0), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (this *MockUserRepository) FindWhere(where entity.User, with ...string) (*entity.User, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) FindManyWhere(where entity.User, with ...string) (*[]entity.User, error) {
	args := this.Called(where, with)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]entity.User), args.Error(1)
}

func (this *MockUserRepository) Create(user entity.User) (*entity.User, error) {
	args := this.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (this *MockUserRepository) Update(user *entity.User) error {
	args := this.Called(user)
	return args.Error(0)
}

type MockHashService struct {
	mock.Mock
}

func (this *MockHashService) HashText(text string, salt string) (string, error) {
	args := this.Called(text, salt)
	return args.String(0), args.Error(1)
}

func (this *MockHashService) Compare(text string, hash string) (bool, error) {
	args := this.Called(text, hash)
	return args.Bool(0), args.Error(1)
}

type MockSessionService struct {
	mock.Mock
}

func (this *MockSessionService) CreateNew(app *entity.App, user *entity.User, request dto.RequestInfo, loginType string) (*entity.Session, error) {
	args := this.Called(app, user, request, loginType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (this *MockSessionService) EncryptSessionToken(sessionId string, token string, secretKey string) (string, error) {
	args := this.Called(sessionId, token, secretKey)
	return args.String(0), args.Error(1)
}

func (this *MockSessionService) DecryptSessionToken(tokenString string, secretKey string) (string, string, error) {
	args := this.Called(tokenString, secretKey)
	return args.String(0), args.String(1), args.Error(2)
}

func (this *MockSessionService) UseSession(session string) error {
	args := this.Called(session)
	return args.Error(0)
}

type MockAuthorizeService struct {
	mock.Mock
}

func (this *MockAuthorizeService) Authorize(app *entity.App, token string, ip string) (*dto.AuthorizeResponse, error) {
	args := this.Called(app, token, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuthorizeResponse), args.Error(1)
}

func (this *MockAuthorizeService) Refresh(app *entity.App, token string, ip string) (*dto.RefreshResponse, error) {
	args := this.Called(app, token, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.RefreshResponse), args.Error(1)
}

func (this *MockAuthorizeService) CreateAuthorizationCredentials(app *entity.App, session *entity.Session) (*services.AuthorizationCredentials, error) {
	args := this.Called(app, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthorizationCredentials), args.Error(1)
}

func (this *MockAuthorizeService) ResetPassword(app *entity.App, payload dto.ResetPasswordPayload) (*entity.User, error) {
	args := this.Called(app, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

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
