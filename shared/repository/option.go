package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Direction string

const (
	Asc  Direction = "ASC"
	Desc Direction = "DESC"
)

// DefaultLimit is applied whenever a listing does not ask for a page size, so
// an unfiltered Find can never pull the whole table by accident.
const DefaultLimit = 10

// Sort is a safe representation of an ORDER BY entry. The column is always
// quoted by gorm (clause.OrderByColumn), so it can never be used to inject SQL.
type Sort struct {
	Column    string
	Direction Direction
}

// Option is the single knob every repository action accepts.
// It carries the transaction, the relations to preload, the page and the
// ordering.
//
// There is deliberately NO raw "where" escape hatch here: query logic that
// cannot be expressed by the typed `where` argument belongs to a dedicated
// repository method, not to the caller.
//
// The zero value means "no transaction, no preloads, first page of
// DefaultLimit rows". A negative Limit is the explicit opt out and drops the
// LIMIT clause — use it only for internal jobs that genuinely need every row.
type Option struct {
	Tx   *Tx
	With []string

	Skip  int
	Limit int
	Sort  []Sort
}

// ResolveOption merges the variadic option into a single value so every
// repository method can treat it as if it were always present.
// Only the first option is read: passing two is not a merge.
func ResolveOption(options ...Option) Option {
	if len(options) == 0 {
		return Option{}
	}

	return options[0]
}

// Paginate returns a copy of the option with a page built from client supplied
// values. Anything not positive is ignored and falls back to the default, so a
// request asking for `limit=-1` cannot disable the limit. Set Limit on the
// struct directly when an internal job really needs every row.
func (this Option) Paginate(skip int, limit int) Option {
	if skip > 0 {
		this.Skip = skip
	}

	if limit > 0 {
		this.Limit = limit
	}

	return this
}

// Size is the page size actually applied: the requested one, or DefaultLimit
// when none was asked for.
func (this Option) Size() int {
	if this.Limit == 0 {
		return DefaultLimit
	}

	return this.Limit
}

// Session returns the transaction when one is provided, otherwise the default client.
func (this Option) Session(client *gorm.DB) *gorm.DB {
	if session := this.Tx.session(); session != nil {
		return session
	}

	return client
}

// Apply adds the preloads, the ordering and the page to a listing query.
// It must never be used on a count query: counting is always done over the
// unpaginated result set.
func (this Option) Apply(query *gorm.DB) *gorm.DB {
	for _, relation := range this.With {
		query = query.Preload(relation)
	}

	for _, sort := range this.Sort {
		if sort.Column == "" {
			continue
		}

		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: sort.Column},
			Desc:   sort.Direction == Desc,
		})
	}

	// NOTE: a negative limit tells gorm to drop the LIMIT clause, which is how
	// "give me everything" is expressed. Zero falls back to DefaultLimit.
	query = query.Limit(this.Size())

	if this.Skip > 0 {
		query = query.Offset(this.Skip)
	}

	return query
}
