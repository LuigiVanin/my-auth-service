package repository

import (
	entity "auth_service/infra/entities"

	"github.com/jmoiron/sqlx"
)

type UserPoolRepository struct {
	client *sqlx.DB
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *sqlx.DB) IUserPoolRepository {
	return &UserPoolRepository{
		client: client,
	}
}

func (r *UserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UserPool, error) {
	return nil, nil
}
