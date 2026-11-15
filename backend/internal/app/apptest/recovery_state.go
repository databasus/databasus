package apptest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// WithRecoveryPause builds an instance in the state a recovery publishes: paused by a completed
// recovery of the local journal until root admin activates it. A test uses it to check what the
// pause blocks and to activate the instance.
func WithRecoveryPause() Option {
	return func(applied *settings) {
		applied.isRecoveryPaused = true
	}
}

// GetPausedRecoveryID is the recovery WithRecoveryPause paused the instance with.
func (a *App) GetPausedRecoveryID() uuid.UUID {
	return a.pausedRecoveryID
}

// MarkStorageRecovered puts a storage into the state recovery normalization leaves a local storage
// in, which the API cannot create.
func (a *App) MarkStorageRecovered(tb testing.TB, storageID uuid.UUID) {
	tb.Helper()

	a.ExecuteSQL(tb, "UPDATE storages SET recovery_validation_status = 'REQUIRES_VALIDATION' WHERE id = ?", storageID)
}

func (a *App) GetDataFolder() string { return a.dataFolder }

// PreparedRecovery is an uploaded recovery that is ready for apply.
type PreparedRecovery struct {
	ID            uuid.UUID
	ArchiveSha256 string
}

// PrepareRecovery records an uploaded recovery as prepared, ready for apply until an hour from the
// application's clock. Preparing one through the API needs a Databasus backup file, which only a
// running backup produces; the journal entry is what such a preparation leaves.
func (a *App) PrepareRecovery(tb testing.TB) PreparedRecovery {
	tb.Helper()

	prepared := PreparedRecovery{ID: uuid.New(), ArchiveSha256: strings.Repeat("ab", 32)}
	suffix := strings.ReplaceAll(prepared.ID.String(), "-", "")[:16]
	now := a.Clock.Now().UTC()

	writeRecoveryJournal(tb, a.recoveryStateDirectory, map[string]any{
		"publicationGeneration":   0,
		"consumedStartupArchives": []string{},
		"operation": map[string]any{
			"id":                    prepared.ID,
			"origin":                "UPLOAD",
			"phase":                 "IMPORTED",
			"archiveSha256":         prepared.ArchiveSha256,
			"sourceRelease":         "dev",
			"snapshotAt":            now.Add(-time.Hour),
			"stagingDatabase":       "databasus_staging_test_" + suffix,
			"previousDatabase":      "databasus_previous_test_" + suffix,
			"isApplyAccepted":       false,
			"createdAt":             now,
			"expiresAt":             now.Add(time.Hour),
			"discardedCleanupCount": 0,
		},
	})

	return prepared
}

// pauseByCompletedRecovery writes what a startup recovery leaves behind: the pause in the database
// and its completion record in the journal.
func pauseByCompletedRecovery(tb testing.TB, databaseDsn, recoveryStateDirectory string) uuid.UUID {
	tb.Helper()

	recoveryID := uuid.New()
	executeOnDatabase(tb, databaseDsn, `UPDATE instancebackups_configs
		SET recovery_paused_operation_id = ?, recovery_paused_at = now()`, recoveryID)

	completedAt := time.Now().UTC()
	writeRecoveryJournal(tb, recoveryStateDirectory, map[string]any{
		"publicationGeneration":   1,
		"consumedStartupArchives": []string{"0000000000000000000000000000000000000000000000000000000000000000"},
		"lastCompleted": map[string]any{
			"operationId":           recoveryID,
			"origin":                "STARTUP",
			"archiveSha256":         "0000000000000000000000000000000000000000000000000000000000000000",
			"sourceRelease":         "dev",
			"snapshotAt":            completedAt.Add(-time.Hour),
			"createdAt":             completedAt,
			"completedAt":           completedAt,
			"discardedCleanupCount": 0,
		},
	})

	return recoveryID
}

func writeRecoveryJournal(tb testing.TB, recoveryStateDirectory string, content map[string]any) {
	tb.Helper()

	journal, err := json.Marshal(content)
	if err != nil {
		tb.Fatalf("encode the recovery journal: %v", err)
	}

	if err := os.MkdirAll(recoveryStateDirectory, 0o700); err != nil {
		tb.Fatalf("create the recovery directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(recoveryStateDirectory, "journal.json"), journal, 0o600); err != nil {
		tb.Fatalf("write the recovery journal: %v", err)
	}
}
