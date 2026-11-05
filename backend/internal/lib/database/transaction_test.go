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

func TestRequireTransaction(t *testing.T) {
	t.Parallel()

	t.Run("refuses calls without a transaction and after detaching one", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, database.RequireTransaction(t.Context()), database.ErrNoTransaction)
		db := openDatabaseWithNotesTable(t)

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, database.RequireTransaction(ctx))
			require.ErrorIs(t, database.RequireTransaction(database.WithoutTransaction(ctx)), database.ErrNoTransaction)

			return nil
		})

		require.NoError(t, err)
	})
}

func TestAfterCommit(t *testing.T) {
	t.Parallel()

	t.Run("runs the hook after the transaction commits", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)
		var notesSeenByHook []string

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, insertNote(ctx, db, "committed"))

			return database.AfterCommit(ctx, func(hookCtx context.Context) {
				notesSeenByHook = listNotesFromContext(hookCtx, t, db)
			})
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"committed"}, notesSeenByHook)
	})

	t.Run("does not run the hook when the transaction rolls back", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)
		isHookRun := false

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, database.AfterCommit(ctx, func(context.Context) { isHookRun = true }))

			return errRollback
		})

		require.ErrorIs(t, err, errRollback)
		assert.False(t, isHookRun)
	})

	t.Run("runs a hook of a nested transaction once, after the outer commit, in order", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)
		transactor := database.NewTransactor(db)
		var hookRuns []string

		err := transactor.Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(
				t,
				database.AfterCommit(ctx, func(context.Context) { hookRuns = append(hookRuns, "outer") }),
			)

			innerErr := transactor.Transaction(ctx, func(innerCtx context.Context) error {
				return database.AfterCommit(innerCtx, func(context.Context) { hookRuns = append(hookRuns, "inner") })
			})
			require.NoError(t, innerErr)
			assert.Empty(t, hookRuns)

			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"outer", "inner"}, hookRuns)
	})

	t.Run("drops the hooks of a nested transaction that fails", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)
		transactor := database.NewTransactor(db)
		var hookRuns []string

		err := transactor.Transaction(t.Context(), func(ctx context.Context) error {
			innerErr := transactor.Transaction(ctx, func(innerCtx context.Context) error {
				require.NoError(t, database.AfterCommit(innerCtx, func(context.Context) {
					hookRuns = append(hookRuns, "inner")
				}))

				return errRollback
			})
			require.ErrorIs(t, innerErr, errRollback)

			return nil
		})

		require.NoError(t, err)
		assert.Empty(t, hookRuns)
	})

	t.Run("refuses outside a transaction", func(t *testing.T) {
		t.Parallel()

		err := database.AfterCommit(t.Context(), func(context.Context) {})

		require.ErrorIs(t, err, database.ErrNoTransaction)
	})
}

func TestWithoutTransaction(t *testing.T) {
	t.Parallel()

	t.Run("writes outside the transaction when the context carried one", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseWithNotesTable(t)

		err := database.NewTransactor(db).Transaction(t.Context(), func(ctx context.Context) error {
			require.NoError(t, insertNote(database.WithoutTransaction(ctx), db, "outside"))

			return errRollback
		})

		require.ErrorIs(t, err, errRollback)
		assert.Equal(t, []string{"outside"}, listNotes(t, db))
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

	return listNotesFromContext(t.Context(), t, db)
}

func listNotesFromContext(ctx context.Context, t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var noteTexts []string
	require.NoError(t, database.Conn(ctx, db).Raw("SELECT text FROM notes ORDER BY text").Scan(&noteTexts).Error)

	return noteTexts
}
