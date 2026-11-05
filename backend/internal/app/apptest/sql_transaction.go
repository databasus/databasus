package apptest

import (
	"testing"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type SQLTransaction struct {
	transaction *gorm.DB
}

func (a *App) BeginSQLTransaction(tb testing.TB) *SQLTransaction {
	tb.Helper()
	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	transaction := db.WithContext(tb.Context()).Begin()
	if transaction.Error != nil {
		closeConnections(db)
		tb.Fatalf("begin a transaction of the test application: %v", transaction.Error)
	}

	tb.Cleanup(func() {
		transaction.Rollback()
		closeConnections(db)
	})
	return &SQLTransaction{transaction: transaction}
}

func (tx *SQLTransaction) ExecuteSQL(tb testing.TB, statement string, values ...any) {
	tb.Helper()
	if err := tx.transaction.Exec(statement, values...).Error; err != nil {
		tb.Fatalf("run %q in a transaction: %v", statement, err)
	}
}

func (tx *SQLTransaction) QuerySQL(tb testing.TB, destination any, statement string, values ...any) {
	tb.Helper()
	if err := tx.transaction.Raw(statement, values...).Scan(destination).Error; err != nil {
		tb.Fatalf("read %q in a transaction: %v", statement, err)
	}
}

func (tx *SQLTransaction) Commit(tb testing.TB) {
	tb.Helper()
	if err := tx.transaction.Commit().Error; err != nil {
		tb.Fatalf("commit a transaction of the test application: %v", err)
	}
}

func (tx *SQLTransaction) Rollback(tb testing.TB) {
	tb.Helper()
	if err := tx.transaction.Rollback().Error; err != nil {
		tb.Fatalf("roll back a transaction of the test application: %v", err)
	}
}
