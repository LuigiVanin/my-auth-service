package repository

import (
	entity "auth_service/infra/entities"
)

type IAppRepository interface {
	FindAppbyIdWithPool(id string) (*entity.App, error)
	Create(app *entity.App) (*entity.App, error)
	FindWhere(where entity.App, with ...string) (*entity.App, error)
}
