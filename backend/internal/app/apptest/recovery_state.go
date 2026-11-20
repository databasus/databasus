package apptest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
)

// WithPublishedRecovery builds an instance in the state a startup recovery publishes: the database
// marked by a completed recovery of the local journal. A test uses it to check what a recovered
// instance does from its first start.
func WithPublishedRecovery() Option {
	return func(applied *settings) {
		applied.isRecoveryPublished = true
	}
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

// markByCompletedRecovery writes what a startup recovery leaves behind: the recovery marker in the
// database and its completion record in the journal.
func markByCompletedRecovery(tb testing.TB, databaseDsn, recoveryStateDirectory string) {
	tb.Helper()

	recoveryID := uuid.New()
	executeOnDatabase(tb, databaseDsn, `UPDATE instancebackups_configs
		SET recovered_by_operation_id = ?, recovered_at = now()`, recoveryID)

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

// NormalizeRecoveredStorages runs the storages step a recovery applies to its staged snapshot over
// the running database, so storages created through the API stand for storages of a recovered
// snapshot.
func (a *App) NormalizeRecoveredStorages(tb testing.TB) {
	tb.Helper()

	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	defer closeConnections(db)

	if _, err := storages.NormalizeRecoveredState(tb.Context(), db); err != nil {
		tb.Fatalf("normalize the recovered storages: %v", err)
	}
}
