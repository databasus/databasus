package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

type BackupCompletionSpec struct {
	Image                string
	ShouldUseIncremental bool
}

func RunBackupCompletionRecovery(t *testing.T, spec BackupCompletionSpec) {
	t.Helper()

	team := setupReplicationOnlyTeam(t, physicalSourceSpec{
		Image: spec.Image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM, Options: nil,
	})
	connection := openConnAt(t, team.Settings.Host, team.Settings.Port)
	_, err := connection.Exec(t.Context(), "ALTER SYSTEM SET autovacuum = off")
	require.NoError(t, err)

	_, err = connection.Exec(t.Context(), "SELECT pg_reload_conf()")
	require.NoError(t, err)

	testwait.Until(t, "the idle source has no automatic transaction writers", func(ctx context.Context) error {
		var isIdle bool
		if err := connection.QueryRow(ctx, "SELECT NOT current_setting('autovacuum')::boolean AND NOT EXISTS(SELECT 1 FROM pg_stat_activity WHERE backend_type='autovacuum worker')").
			Scan(&isIdle); err != nil {
			return err
		}

		if !isIdle {
			return fmt.Errorf("autovacuum is not disabled and idle")
		}

		return nil
	})

	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-full", Payload: "included in the full backup"})

	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	phases := []string{"before-full"}
	if spec.ShouldUseIncremental {
		insertMarker(
			t,
			connection,
			markerRow{Phase: "before-incremental", Payload: "included in the incremental backup"},
		)

		chain = team.buildIncremental(
			t,
			connection,
			incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1},
		)
		phases = append(phases, "before-incremental")
	}

	completed := chain[len(chain)-1]
	require.NotNil(t, completed.CompletedAt)
	team.waitForWalWithoutLaterTransactions(t, connection, completed)

	statistics, err := team.Owner.Client.GetWalStreamStatisticsWithResponse(t.Context(), team.Database.Id,
		&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriodHOUR})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statistics.StatusCode(), "%s", statistics.Body)
	require.NotNil(t, statistics.JSON200)
	require.NotNil(t, statistics.JSON200.LatestRestorableAt)
	require.Equal(t, completed.CompletedAt.UTC(), statistics.JSON200.LatestRestorableAt.UTC())

	targetTime := statistics.JSON200.LatestRestorableAt
	bundle := team.downloadRestoreBundle(t, targetTime)
	target := prepareRestoreTarget(t, spec.Image)
	reconstructCluster(
		t,
		target,
		restoreSpec{App: team.App, Image: spec.Image, BundlePath: bundle, TargetTime: targetTime},
	)

	startRestoredCluster(t, target, spec.Image)

	require.ElementsMatch(t, phases, queryRestoredMarkerRows(t, target))

	restored := openConnAt(t, target.Host(), target.MappedPort())
	insertMarker(t, restored, markerRow{Phase: "after-recovery", Payload: "the recovered cluster accepts writes"})

	target.Exec(t, "sh", "-c", "test ! -d /restore/bundle/wal || test -z \"$(ls -A /restore/bundle/wal)\"")
}

func (team Team) waitForWalWithoutLaterTransactions(t *testing.T, connection *pgx.Conn, completed backupRow) {
	t.Helper()

	var archived struct {
		SegmentCount        int
		HasLaterTransaction bool
	}
	testwait.Until(t, "transaction-free WAL after backup completion is published", func(ctx context.Context) error {
		if _, err := connection.Exec(ctx, "CHECKPOINT; SELECT pg_switch_wal()"); err != nil {
			return err
		}

		team.App.QuerySQL(
			t,
			&archived,
			"SELECT COUNT(*) AS segment_count, COALESCE(BOOL_OR(last_transaction_at > ?), false) AS has_later_transaction FROM backups_physical_wal_segments WHERE database_id=? AND timeline_id=? AND file_name IS NOT NULL AND end_lsn > ?::pg_lsn",
			completed.CompletedAt,
			team.Database.Id,
			completed.TimelineID,
			completed.StopLSN,
		)
		if archived.SegmentCount == 0 {
			return fmt.Errorf("no archived WAL extends past backup %s", completed.ID)
		}

		return nil
	})

	require.False(t, archived.HasLaterTransaction)
}
