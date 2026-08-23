package repository

import (
	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
)

type UserPoolRepository struct {
	repo.BaseRepository[entity.UsersPool, dto.UserPoolUpdateDao]
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *gorm.DB) *UserPoolRepository {
	return &UserPoolRepository{
		BaseRepository: repo.NewBaseRepository[entity.UsersPool, dto.UserPoolUpdateDao](client),
	}
}
