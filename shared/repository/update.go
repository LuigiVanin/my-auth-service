package repository

import "reflect"

// NoUpdate marks an entity that is not updatable through the repository, such
// as a read only reference table: BaseRepository[entity.X, repo.NoUpdate].
// Its Update method always resolves to an empty map and touches nothing.
type NoUpdate struct{}

// IUpdateDao lets a dao override the default reflection based conversion when
// it needs to normalize values (lowercase an email, stamp a column, ...).
type IUpdateDao interface {
	ToUpdateMap() map[string]any
}

// BuildUpdateMap turns an update dao into the map gorm expects.
//
// Why a map and not the dao struct itself: gorm only auto fills AutoUpdateTime
// columns (`UpdatedAt`) on the map branch of ConvertToAssignments. When the
// destination is a struct whose schema differs from the model, gorm walks the
// model columns and looks them up in the dao schema, so a dao without an
// `UpdatedAt` field silently leaves `updated_at` untouched.
//
// Nil pointers are skipped (field not updated), non nil pointers are
// dereferenced, so `false`, `0` and `""` are written normally.
// Non pointer fields are always sent.
func BuildUpdateMap(dao any) map[string]any {
	if custom, ok := dao.(IUpdateDao); ok {
		return custom.ToUpdateMap()
	}

	values := map[string]any{}

	reflected := reflect.Indirect(reflect.ValueOf(dao))
	if reflected.Kind() != reflect.Struct {
		return values
	}

	daoType := reflected.Type()

	for index := range daoType.NumField() {
		field := daoType.Field(index)

		if !field.IsExported() {
			continue
		}

		value := reflected.Field(index)

		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}

			value = value.Elem()
		}

		// NOTE: the key is the entity field name, gorm resolves it to the column
		// through the model schema (LookUpField), so no naming strategy is duplicated here.
		values[field.Name] = value.Interface()
	}

	return values
}
