package services

import (
	entity "auth_service/infra/entities"
)

type IUserPoolService interface {
	Create(name string, ownerUserId *uint) (*entity.UsersPool, error)
	FindById(id string) (*entity.UsersPool, error)
}
