package mock

import (
	"auth_service/shared/storage"

	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

var _ storage.IStorage = &MockStorage{}

func (this *MockStorage) PresignUpload(key string, contentType string, policy storage.Policy) (*storage.UploadGrant, error) {
	args := this.Called(key, contentType, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadGrant), args.Error(1)
}

func (this *MockStorage) Head(key string) (*storage.Object, error) {
	args := this.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Object), args.Error(1)
}

func (this *MockStorage) Copy(source string, destination string) error {
	args := this.Called(source, destination)
	return args.Error(0)
}

func (this *MockStorage) Delete(key string) error {
	args := this.Called(key)
	return args.Error(0)
}
