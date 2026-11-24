package user_pool_test

import (
	"testing"

	dto "auth_service/app/modules/core/user_pool/models"
	upr "auth_service/app/modules/core/user_pool/repository"
	"auth_service/app/modules/core/user_pool/services"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"
	"auth_service/shared/storage"
	mock "auth_service/tests/modules/mock"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// listWith runs the listing and returns the option and search the repository saw.
func listWith(t *testing.T, query *dto.UserPoolListQuery) (repo.Option, upr.UserPoolSearch) {
	repository := new(mock.MockUserPoolRepository)
	service := services.NewUserPoolService(
		repository,
		new(mock.MockProfileService),
		new(mock.MockCipherService),
		new(mock.MockTransactionManager),
		zap.NewNop(),
		storage.NewUrlResolver("https://cdn.test"),
	)

	var seen repo.Option
	var search upr.UserPoolSearch
	repository.On("FindSearch", testifymock.Anything, testifymock.Anything).
		Run(func(args testifymock.Arguments) {
			search = args.Get(0).(upr.UserPoolSearch)
			seen = args.Get(1).([]repo.Option)[0]
		}).
		Return([]entity.UsersPool{}, nil)
	repository.On("FindSearchCount", testifymock.Anything, testifymock.Anything).Return(int64(0), nil)

	_, err := service.List(&entity.Organization{ID: organizationId}, query)
	assert.NoError(t, err)

	return seen, search
}

func TestUserPoolList_OrderAlwaysEndsOnTheId(t *testing.T) {
	cases := map[string]repo.Sort{
		"":        {Column: "updated_at", Direction: repo.Desc},
		"updated": {Column: "updated_at", Direction: repo.Desc},
		"name":    {Column: "name", Direction: repo.Asc},
		"signups": {Column: "users_count", Direction: repo.Desc},
		"created": {Column: "created_at", Direction: repo.Asc},
	}

	for sort, first := range cases {
		option, _ := listWith(t, &dto.UserPoolListQuery{Sort: sort})
		assert.Equal(t, []repo.Sort{first, {Column: "id", Direction: repo.Asc}}, option.Sort, sort)
	}
}

func TestUserPoolList_PassesCeilingAndScopeThrough(t *testing.T) {
	_, search := listWith(t, &dto.UserPoolListQuery{Ceiling: "broad", Scope: "system"})

	assert.Equal(t, organizationId, search.OrganizationId)
	assert.Equal(t, "broad", search.Ceiling)
	assert.Equal(t, "system", search.Scope)
}
