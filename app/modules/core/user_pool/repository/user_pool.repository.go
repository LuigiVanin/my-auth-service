package repository

import (
	"encoding/json"

	dto "auth_service/app/modules/core/user_pool/models"
	entity "auth_service/infra/entities"
	repo "auth_service/shared/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// searchScope is the single definition of the listing predicate, shared by
// FindSearch and FindSearchCount so the two cannot drift apart.
func searchScope(search UserPoolSearch) func(*gorm.DB) *gorm.DB {
	return func(query *gorm.DB) *gorm.DB {
		query = query.Where("users_pool.organization_id = ?", search.OrganizationId)

		if search.Name != "" {
			query = query.Where("users_pool.name ILIKE ?", "%"+search.Name+"%")
		}

		return query
	}
}

func (this *UserPoolRepository) FindSearch(search UserPoolSearch, options ...repo.Option) ([]entity.UsersPool, error) {
	result := []entity.UsersPool{}

	err := this.ListQuery(options...).Scopes(searchScope(search)).Find(&result).Error

	return result, err
}

func (this *UserPoolRepository) FindSearchCount(search UserPoolSearch, options ...repo.Option) (int64, error) {
	var count int64

	err := this.Query(options...).Scopes(searchScope(search)).Count(&count).Error

	return count, err
}

// A read modify write would lose a concurrent signup, and `count + 1` is an
// expression the pointer based update dao cannot carry, so the increment is a
// query and lives here. UpdateColumn, so `updated_at` keeps meaning "the pool
// was configured" rather than "somebody signed up".
func (this *UserPoolRepository) IncrementUsersCount(id string, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		UpdateColumn("users_count", gorm.Expr("users_count + ?", 1))

	return result.RowsAffected, result.Error
}

// FindOneForUpdate reads the row and holds it until the transaction ends. It is
// what makes the read-modify-write of the tracking document safe: two concurrent
// signups into the same pool serialize here instead of losing one of the two.
//
// Query and not ListQuery: the latter would apply the default page LIMIT to a
// single row read.
func (this *UserPoolRepository) FindOneForUpdate(id string, options ...repo.Option) (*entity.UsersPool, error) {
	query := this.Query(options...).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id)

	return repo.FirstOrNil[entity.UsersPool](query)
}

// WriteTracking is the only way the tracking column is written: it is absent from
// the update dao, so no payload can reach it. UpdateColumn, so a signup does not
// move `updated_at` of the pool.
func (this *UserPoolRepository) WriteTracking(id string, document json.RawMessage, options ...repo.Option) (int64, error) {
	result := this.Query(options...).
		Where("id = ?", id).
		UpdateColumn("tracking", document)

	return result.RowsAffected, result.Error
}
