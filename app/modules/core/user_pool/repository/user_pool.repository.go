package repository

import (
	e "auth_service/app/errors"
	"auth_service/app/modules/utils/cipher"
	entity "auth_service/infra/entities"

	"gorm.io/gorm"
)

type UserPoolRepository struct {
	client        *gorm.DB
	cipherService cipher.ICipherService
}

var _ IUserPoolRepository = &UserPoolRepository{}

func NewUserPoolRepository(client *gorm.DB, cipherService cipher.ICipherService) *UserPoolRepository {
	return &UserPoolRepository{
		client:        client,
		cipherService: cipherService,
	}
}

func (r *UserPoolRepository) FindByAppIdAndPoolId(id string, appId string) (*entity.UsersPool, error) {
	// Implementation was empty in original code.
	// Preserving empty implementation but complying with GORM structure.
	return nil, nil
}

func (this *UserPoolRepository) Create(userPool *entity.UsersPool) (*entity.UsersPool, error) {
	err := this.client.Create(userPool).Error
	if err != nil {
		return nil, err
	}

	pool, err := this.FindById(userPool.ID)

	if err != nil {
		return nil, err
	}

	pool.PublicKey, err = this.cipherService.EncryptUuidIntoToken(pool.ID)

	if err != nil {
		return nil, err
	}

	err = this.client.Save(pool).Error

	if err != nil {
		return nil, e.ThrowInternalServerError("Unable to update app after creation")
	}

	return userPool, nil
}

func (this *UserPoolRepository) FindById(id string) (*entity.UsersPool, error) {
	var result entity.UsersPool
	err := this.client.Where("id = ?", id).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}
