package migrations_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/migrations"
)

func TestVerificationFailureDiagnosticsMigration(t *testing.T) {
	t.Parallel()
	t.Run("preserves historical failures and enforces output bounds when migrated and rolled back", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261106131020_verification_failure_diagnostics.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		fixture := insertVerificationFixture(t, db)
		verificationID := uuid.New()
		execute(t, db, `INSERT INTO verification_runs(id,database_id,backup_id,status,trigger,fail_message)
   VALUES(?,?,?,'FAILED','MANUAL','original failure')`, verificationID, fixture.databaseID, fixture.backupID)

		applyAllMigrations(t, db, migrationName)

		var diagnostic struct {
			Output      *string
			IsTruncated bool
			FailMessage string
		}
		require.NoError(t, db.WithContext(t.Context()).Raw(`SELECT failure_diagnostic_output AS output,
   is_failure_diagnostic_truncated AS is_truncated,fail_message FROM verification_runs WHERE id=?`, verificationID).Scan(&diagnostic).Error)
		assert.Nil(t, diagnostic.Output)
		assert.False(t, diagnostic.IsTruncated)
		assert.Equal(t, "original failure", diagnostic.FailMessage)

		for _, invalidDiagnostic := range []struct {
			output      *string
			isTruncated bool
		}{
			{output: nil, isTruncated: true},
			{output: new(""), isTruncated: false},
			{output: new(strings.Repeat("a", 65537)), isTruncated: false},
			{output: new(strings.Repeat("\u754c", 22000)), isTruncated: false},
		} {
			err := db.WithContext(t.Context()).Exec(`UPDATE verification_runs SET failure_diagnostic_output=?,
    is_failure_diagnostic_truncated=? WHERE id=?`, invalidDiagnostic.output, invalidDiagnostic.isTruncated, verificationID).Error
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ck_verification_runs_failure_diagnostic")
		}

		execute(
			t,
			db,
			"UPDATE verification_runs SET failure_diagnostic_output=?,is_failure_diagnostic_truncated=true WHERE id=?",
			strings.Repeat("a", 65536),
			verificationID,
		)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)

		_, err = provider.ApplyVersion(t.Context(), 20261106131020, false)
		require.NoError(t, err)

		var failureSummary string
		require.NoError(
			t,
			db.WithContext(t.Context()).
				Raw("SELECT fail_message FROM verification_runs WHERE id=?", verificationID).
				Scan(&failureSummary).
				Error,
		)
		assert.Equal(t, "original failure", failureSummary)
	})
}
