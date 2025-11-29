package repository

import (
	entity "auth_service/infra/entities"

	"github.com/jmoiron/sqlx"
)

type UserPoolRepository struct {
	db *sqlx.DB
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(db *sqlx.DB) IUserPoolRepository {
	return &UserPoolRepository{
		db: db,
	}
}

func (r *UserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UserPool, error) {
	return nil, nil
}
