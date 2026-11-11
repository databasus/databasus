package restoring_test

import (
	"net/http"
	"testing"
	"time"
)

func TestSelectPhysicalBackupCompletion(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name                   string
		hasTransactionEvidence bool
		transactionOffset      time.Duration
		shouldReplayWal        bool
	}{
		{name: "ships only the backup when following WAL has no transaction evidence"},
		{name: "ships only the backup when WAL transactions precede its completion", hasTransactionEvidence: true, transactionOffset: -time.Minute},
		{name: "retains WAL when a transaction confirms the exact completion target", hasTransactionEvidence: true, transactionOffset: time.Microsecond, shouldReplayWal: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			timeline := newRestoreTimeline()
			timeline.base = timeline.base.Truncate(time.Microsecond)
			completion := timeline.at(0)
			full := catalog.seedRestoreFull(t, restoreFullSpec{StopLSN: lsnAt(1), CompletedAt: completion})
			segment := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 1, ReceivedAt: timeline.at(15)})
			var transactionTime *time.Time
			if scenario.hasTransactionEvidence {
				transactionTime = new(completion.Add(scenario.transactionOffset))
			}

			catalog.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET last_transaction_at=? WHERE id=?",
				transactionTime, segment.ID)

			entries := catalog.downloadRestore(t, restoreRequest{TargetTime: &completion})

			if scenario.shouldReplayWal {
				assertRestoreArtifacts(t, entries, full.ID, nil, 1)
			} else {
				assertRestoreArtifacts(t, entries, full.ID, nil)
			}

			latest := catalog.downloadRestore(t, restoreRequest{})

			assertRestoreArtifacts(t, latest, full.ID, nil, 1)

			catalog.downloadRestore(t, restoreRequest{
				TargetTime: new(completion.Add(2 * time.Microsecond)), StatusCode: http.StatusConflict,
			})
		})
	}
}
