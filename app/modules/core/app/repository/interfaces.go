package repository

import (
	entity "auth_service/infra/entities"
)

type IAppRepository interface {
	FindAppbyIdWithPool(id string) (*entity.App, error)
	Create(app *entity.App) (*entity.App, error)
	FindWhere(query entity.App, with ...string) (*entity.App, error)

	FindManyWhere(query any, args []any, with ...string) (*[]entity.App, error)

	FindManyWhereAndCount(query any, args []any, skip int, limit int, with ...string) (*[]entity.App, int64, error)
}
