package repository

import entity "auth_service/infra/entities"

type IAppRepository interface {
	FindAppbyIdWithPool(id string) (*entity.AppWithPool, error)
}
