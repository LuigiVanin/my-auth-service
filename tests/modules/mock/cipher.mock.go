package mock

import (
	"auth_service/app/modules/utils/cipher"

	"github.com/stretchr/testify/mock"
)

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
