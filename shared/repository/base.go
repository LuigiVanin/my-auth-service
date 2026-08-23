package repository

import (
	"errors"

	"gorm.io/gorm"
)

// IsNotFound reports whether err is gorm's "record not found", so no service
// or repository outside this package needs to import gorm for it.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// FirstOrNil runs First on the query and folds "record not found" into
// (nil, nil), matching the FindOne contract. Dedicated single row queries
// written by concrete repositories should use it instead of re-handling
// ErrRecordNotFound.
func FirstOrNil[T any](query *gorm.DB) (*T, error) {
	var result T

	if err := query.First(&result).Error; err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return &result, nil
}

// BaseRepository implements the whole generic contract once.
// T is the entity, U is the update dao of that entity.
type BaseRepository[T any, U any] struct {
	client *gorm.DB
}

func NewBaseRepository[T any, U any](client *gorm.DB) BaseRepository[T, U] {
	return BaseRepository[T, U]{client: client}
}

// Query is the building block for dedicated queries written by concrete
// repositories: it is already scoped to the entity table and to the transaction.
func (this *BaseRepository[T, U]) Query(options ...Option) *gorm.DB {
	var model T

	return ResolveOption(options...).Session(this.client).Model(&model)
}

// ListQuery is Query plus preloads, ordering and pagination.
func (this *BaseRepository[T, U]) ListQuery(options ...Option) *gorm.DB {
	option := ResolveOption(options...)

	return option.Apply(this.Query(option))
}

func (this *BaseRepository[T, U]) Find(where T, options ...Option) ([]T, error) {
	result := []T{}

	err := this.ListQuery(options...).Where(where).Find(&result).Error

	return result, err
}

// FindCount counts the rows matching `where`, ignoring pagination and preloads.
func (this *BaseRepository[T, U]) FindCount(where T, options ...Option) (int64, error) {
	var count int64

	err := this.Query(options...).Where(where).Count(&count).Error

	return count, err
}

// FindOne returns nil, nil when nothing matches, so services never have to
// import gorm to check for ErrRecordNotFound.
func (this *BaseRepository[T, U]) FindOne(where T, options ...Option) (*T, error) {
	return FirstOrNil[T](this.ListQuery(options...).Where(where))
}

func (this *BaseRepository[T, U]) Create(value T, options ...Option) (*T, error) {
	err := ResolveOption(options...).Session(this.client).Create(&value).Error

	return &value, err
}

// Update applies the non nil fields of the dao to every row matching `where`.
// An empty `where` is rejected by gorm itself (ErrMissingWhereClause).
func (this *BaseRepository[T, U]) Update(where T, dao U, options ...Option) (int64, error) {
	values := BuildUpdateMap(dao)

	if len(values) == 0 {
		return 0, nil
	}

	result := this.Query(options...).Where(where).Updates(values)

	return result.RowsAffected, result.Error
}

func (this *BaseRepository[T, U]) Delete(where T, options ...Option) (int64, error) {
	var model T

	result := this.Query(options...).Where(where).Delete(&model)

	return result.RowsAffected, result.Error
}
