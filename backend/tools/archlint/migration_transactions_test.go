package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationTransactions(t *testing.T) {
	t.Parallel()

	t.Run("fails when a migration runs without a transaction", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"migrations/20250101000000_add_index.sql": "-- +goose NO TRANSACTION\n-- +goose Up\n" +
				"CREATE INDEX CONCURRENTLY idx_backups_created_at ON backups (created_at);\n",
		})

		requireSingleViolation(t, violations, violationSpec{
			position:     "migrations/20250101000000_add_index.sql:1",
			rule:         "migration-transaction",
			messageParts: []string{"NO TRANSACTION"},
		})
	})

	t.Run("passes when a migration runs in its transaction", func(t *testing.T) {
		t.Parallel()
		violations := checkFixture(t, map[string]string{
			"migrations/20250101000000_add_index.sql": "-- +goose Up\n" +
				"CREATE INDEX idx_backups_created_at ON backups (created_at);\n",
		})

		assert.Empty(t, violations)
	})
}
