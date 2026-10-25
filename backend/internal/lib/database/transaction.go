package database

import (
	"context"

	"gorm.io/gorm"
)

type transactionKey struct{}

// Transactor opens transactions on the pool it was built with. Services take it through Deps, so
// they open a transaction without holding the pool themselves.
type Transactor struct {
	db *gorm.DB
}

func NewTransactor(db *gorm.DB) *Transactor {
	return &Transactor{db: db}
}

// Transaction runs fn in a transaction that the context passed to fn carries. Every query made
// through Conn with that context joins it. The transaction commits when fn returns nil and rolls
// back otherwise. Called inside another transaction, it runs fn under a savepoint of that one.
func (t *Transactor) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return Conn(ctx, t.db).Transaction(func(transaction *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionKey{}, transaction))
	})
}

// Conn returns the transaction the context carries, or db bound to the context outside a
// transaction.
func Conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	transaction, isInTransaction := ctx.Value(transactionKey{}).(*gorm.DB)
	if isInTransaction {
		return transaction
	}

	return db.WithContext(ctx)
}
