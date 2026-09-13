package repository

import (
	"encoding/json"

	dto "auth_service/app/modules/core/user/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	repo.BaseRepository[entity.User, dto.UserUpdateDao]
}

var _ IUserRepository = &UserRepository{}

func NewUserRepository(client *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: repo.NewBaseRepository[entity.User, dto.UserUpdateDao](client),
	}
}

// searchScope is the single definition of the listing predicate, shared by
// FindSearch and FindSearchCount so the two cannot drift apart.
func searchScope(search UserSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		query = query.Where("users.users_pool_id = ?", search.UsersPoolId)

		if search.Name != "" {
			query = query.Where("users.name ILIKE ?", "%"+search.Name+"%")
		}

		if search.Email != "" {
			query = query.Where("users.email ILIKE ?", "%"+search.Email+"%")
		}

		return query
	}
}

func (this *UserRepository) FindSearch(search UserSearch, options ...repo.Option) ([]entity.User, error) {
	result := []entity.User{}

	err := this.ListQuery(options...).Scopes(searchScope(search)).Find(&result).Error

	return result, err
}

func (this *UserRepository) FindSearchCount(search UserSearch, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).Scopes(searchScope(search)).Count(&count).Error

	return count, err
}

// FindOneForUpdate reads the row and holds it until the transaction ends, so two
// concurrent logins of the same user cannot lose one another's event.
//
// Query and not ListQuery: the latter would apply the default page LIMIT to a
// single row read.
func (this *UserRepository) FindOneForUpdate(id uint, options ...repo.Option) (*entity.User, error) {
	query := this.Query(options...).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id)

	return repo.FirstOrNil[entity.User](query)
}

// WriteTracking is the only way the tracking column is written: it is absent from
// the update dao, so no payload can reach it. UpdateColumn, so a login does not
// move `updated_at` of the user.
func (this *UserRepository) WriteTracking(id uint, document json.RawMessage, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		UpdateColumn("tracking", document)

	return result.RowsAffected, result.Error
}
