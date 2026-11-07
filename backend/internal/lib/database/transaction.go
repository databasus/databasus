package database

import (
	"context"

	"gorm.io/gorm"
)

type (
	transactionKey      struct{}
	afterCommitHooksKey struct{}
)

type afterCommitHooks struct {
	hooks []func(ctx context.Context)
}

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
//
// The after-commit hooks registered in fn run after the outermost transaction commits. A nested
// transaction hands its hooks to the enclosing one when it succeeds, and drops them when it fails.
func (t *Transactor) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	registered := &afterCommitHooks{}

	err := Conn(ctx, t.db).Transaction(func(transaction *gorm.DB) error {
		transactionCtx := context.WithValue(ctx, transactionKey{}, transaction)

		return fn(context.WithValue(transactionCtx, afterCommitHooksKey{}, registered))
	})
	if err != nil {
		return err
	}

	enclosing, isNested := ctx.Value(afterCommitHooksKey{}).(*afterCommitHooks)
	if isNested && enclosing != nil {
		enclosing.hooks = append(enclosing.hooks, registered.hooks...)

		return nil
	}

	committedCtx := context.WithoutCancel(WithoutTransaction(ctx))
	for _, hook := range registered.hooks {
		hook(committedCtx)
	}

	return nil
}

// AfterCommit registers hook to run once the transaction ctx carries commits, for work outside the
// database that must not happen when the transaction rolls back. The hook gets a context without
// the transaction that is not cancelled with ctx. Outside a transaction it refuses with
// ErrNoTransaction, because nothing would guarantee that the work it follows is kept.
func AfterCommit(ctx context.Context, hook func(ctx context.Context)) error {
	registered, isInTransaction := ctx.Value(afterCommitHooksKey{}).(*afterCommitHooks)
	if !isInTransaction || registered == nil {
		return ErrNoTransaction
	}

	registered.hooks = append(registered.hooks, hook)

	return nil
}

func Conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	transaction, isInTransaction := ctx.Value(transactionKey{}).(*gorm.DB)
	if isInTransaction {
		return transaction.WithContext(ctx)
	}

	return db.WithContext(ctx)
}

func RequireTransaction(ctx context.Context) error {
	transaction, isInTransaction := ctx.Value(transactionKey{}).(*gorm.DB)
	if !isInTransaction || transaction == nil {
		return ErrNoTransaction
	}

	return nil
}

// WithoutTransaction returns ctx without the transaction it carries, keeping its other values, so
// work that outlives the transaction, such as a delivery started inside it, queries the pool.
func WithoutTransaction(ctx context.Context) context.Context {
	withoutTransaction := context.WithValue(ctx, transactionKey{}, nil)

	return context.WithValue(withoutTransaction, afterCommitHooksKey{}, nil)
}
