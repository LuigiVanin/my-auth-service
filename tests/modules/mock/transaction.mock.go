package mock

import (
	repo "auth_service/shared/repository"

	"github.com/stretchr/testify/mock"
)

// MockTransactionManager hands out a zero repo.Tx: it carries no gorm client, so
// Commit and Rollback are the no ops they are documented to be and the option a
// service threads through its repositories is still comparable by pointer.
type MockTransactionManager struct {
	mock.Mock
}

var _ repo.ITransactionManager = &MockTransactionManager{}

func (this *MockTransactionManager) Tx() (*repo.Tx, error) {
	args := this.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Tx), args.Error(1)
}
