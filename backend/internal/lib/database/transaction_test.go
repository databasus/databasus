package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

var errRollback = errors.New("roll back")

func TestTransaction(t *testing.T) {
	t.Parallel()

	t.Run("keeps writes made through Conn when fn returns nil", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			return insertNote(ctx, db, "kept")
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"kept"}, listNotes(t, db))
	})

	t.Run("discards writes made through Conn when fn returns an error", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, insertNote(ctx, db, "discarded"))

			return errRollback
		})

		require.ErrorIs(t, err, errRollback)
		assert.Empty(t, listNotes(t, db))
	})

	t.Run("discards only the inner writes when a nested transaction fails", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)
		transactor := database.NewTransactor(db)

		err := transactor.Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, insertNote(ctx, db, "outer"))

			innerErr := transactor.Transaction(ctx, func(innerCtx context.Context) error {
				require.NoError(t, insertNote(innerCtx, db, "inner"))

				return errRollback
			})
			require.ErrorIs(t, innerErr, errRollback)

			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"outer"}, listNotes(t, db))
	})
}

func TestConn(t *testing.T) {
	t.Parallel()

	t.Run("writes at once without a transaction in the context", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)

		require.NoError(t, insertNote(t.Context(), db, "immediate"))

		assert.Equal(t, []string{"immediate"}, listNotes(t, db))
	})
}

func openDatabaseWithNotesTable(t *testing.T) *gorm.DB {
	t.Helper()

	db := openTestDatabase(t)
	require.NoError(t, db.WithContext(t.Context()).Exec("CREATE TABLE notes (text TEXT NOT NULL)").Error)

	return db
}

func insertNote(ctx context.Context, db *gorm.DB, text string) error {
	return database.Conn(ctx, db).Exec("INSERT INTO notes (text) VALUES (?)", text).Error
}

func listNotes(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var noteTexts []string
	require.NoError(t, db.WithContext(t.Context()).Raw("SELECT text FROM notes ORDER BY text").Scan(&noteTexts).Error)

	return noteTexts
}
