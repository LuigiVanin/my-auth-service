package services

import (
	e "auth_service/app/errors"
	dto "auth_service/app/modules/core/user_pool/models"
	upr "auth_service/app/modules/core/user_pool/repository"
	"auth_service/app/modules/utils/cipher"
	entity "auth_service/infra/entities"
)

type UserPoolService struct {
	userPoolRepository upr.IUserPoolRepository
	cipherService      cipher.ICipherService
}

var _ IUserPoolService = &UserPoolService{}

func NewUserPoolService(userPoolRepository upr.IUserPoolRepository, cipherService cipher.ICipherService) *UserPoolService {
	return &UserPoolService{
		userPoolRepository: userPoolRepository,
		cipherService:      cipherService,
	}
}

// Create inserts the pool and stamps its public key, which is derived from the
// id and therefore only knowable after the insert.
//
// NOTE: this used to live inside UserPoolRepository, which meant the repository
// depended on the cipher service and rewrote the whole row with Save. Deriving
// the key is business logic, so it belongs here; the repository now only writes
// the one column that changed.
func (this *UserPoolService) Create(name string, ownerUserId *uint) (*entity.UsersPool, error) {
	pool, err := this.userPoolRepository.Create(entity.UsersPool{
		Name:        name,
		OwnerUserId: ownerUserId,
	})

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to create users pool")
	}

	publicKey, err := this.cipherService.EncryptUuidIntoToken(pool.ID)

	if err != nil {
		return nil, e.ThrowInternalServerError("Failed to encrypt the users pool public key")
	}

	if _, err := this.userPoolRepository.Update(
		entity.UsersPool{ID: pool.ID},
		dto.UserPoolUpdateDao{PublicKey: &publicKey},
	); err != nil {
		return nil, e.ThrowInternalServerError("Unable to update users pool after creation")
	}

	pool.PublicKey = publicKey

	return pool, nil
}

// FindById returns nil, nil when the pool does not exist.
func (this *UserPoolService) FindById(id string) (*entity.UsersPool, error) {
	return this.userPoolRepository.FindOne(entity.UsersPool{ID: id})
}
