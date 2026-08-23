package repository

import "gorm.io/gorm"

// ITransactionManager opens transactions. It is a singleton in the DI
// container, so a service never has to receive the raw *gorm.DB client.
//
// Only reach for a transaction when a single unit of work performs more than
// one mutation. Read only flows and single mutations do not need one: gorm
// already wraps every isolated Create/Update/Delete in its own transaction
// (SkipDefaultTransaction is false), so opening one by hand around a single
// write only adds a BEGIN/COMMIT round trip.
type ITransactionManager interface {
	Tx() (*Tx, error)
}

// Tx is an open transaction, handed to repositories through Option.
type Tx struct {
	db   *gorm.DB
	done bool
}

// session is nil safe: a zero Option simply means "no transaction".
func (this *Tx) session() *gorm.DB {
	if this == nil {
		return nil
	}

	return this.db
}

// Commit persists everything written through this transaction.
func (this *Tx) Commit() error {
	if this == nil || this.db == nil || this.done {
		return nil
	}

	this.done = true

	return this.db.Commit().Error
}

// Rollback discards everything written through this transaction. It is a no op
// once the transaction is finished, so `defer tx.Rollback()` right after
// opening it is always safe and is what keeps an early return from leaking
// the connection.
func (this *Tx) Rollback() error {
	if this == nil || this.db == nil || this.done {
		return nil
	}

	this.done = true

	return this.db.Rollback().Error
}

type TransactionManager struct {
	client *gorm.DB
}

var _ ITransactionManager = &TransactionManager{}

func NewTransactionManager(client *gorm.DB) *TransactionManager {
	return &TransactionManager{client: client}
}

// Tx opens a transaction. The error must be handled: when BEGIN fails gorm
// returns a handle still bound to the pool, which would silently run every
// statement outside of a transaction.
func (this *TransactionManager) Tx() (*Tx, error) {
	tx := this.client.Begin()

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &Tx{db: tx}, nil
}
