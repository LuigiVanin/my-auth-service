package repository

import entity "auth_service/infra/entities"

type IUserPoolRepository interface {
	FindByAppIdAndPoolId(id string, appId string) (*entity.UserPool, error)
}
