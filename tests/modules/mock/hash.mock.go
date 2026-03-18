package mock

import (
	"github.com/stretchr/testify/mock"
)

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
