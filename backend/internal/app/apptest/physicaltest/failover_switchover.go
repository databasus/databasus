package physicaltest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func RunFailoverSwitchovers(t *testing.T, manager FailoverManager) {
	t.Helper()

	source := manager.startSource(t, failoverSourceSpec{})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	chain = source.Team.buildIncremental(
		t,
		connection,
		incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1},
	)
	obsoleteFull := chain[0]
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:   apptest.PhysicalStartRequest(source.Team.Database.Id, "FULL"),
			Status: http.StatusAccepted,
		},
	)
	chain = []backupRow{source.waitForNewFull(t, obsoleteFull.ID)}
	firstRetainedFull := chain[0]
	oldPoint := source.captureRecoveryPoint(t, chain, "before-switches")
	source.restoreRecoveryPoint(t, oldPoint)
	var newPoint recoveryPoint
	var gaps []time.Time

	for switchNumber := 1; switchNumber <= 2; switchNumber++ {
		previous := source.Cluster.GetIdentity(t)
		delayedWal := source.delayWalPublication(t, previous.Timeline)

		promoted := source.Cluster.Switchover(t)
		require.NotEqual(t, previous.Node, promoted.Node)
		require.Greater(t, promoted.Timeline, previous.Timeline)
		connection = source.connect(t)
		insertMarker(
			t,
			connection,
			markerRow{Phase: fmt.Sprintf("after-switch-%d", switchNumber), Payload: "committed on promoted node"},
		)
		gap := sourceTime(t, connection)
		gaps = append(gaps, gap)
		source.publishDelayedWal(t, delayedWal, gap.Add(time.Second))

		source.requireUnreachableTarget(t, gap)
		source.Team.triggerIncremental(t)
		source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
		replacement := source.waitForNewFull(t, chain[0].ID)
		require.Equal(t, promoted.Timeline, replacement.TimelineID)
		source.requireSingleReplacement(t, fullReplacement{Preceding: chain[0].ID, Replacement: replacement.ID})
		chain = []backupRow{replacement}
		source.Team.waitForTimelineHistory(t, promoted.Timeline)
		source.Team.requirePublishedTimelineHistory(t, connection)
		source.Team.requireRestrictedPromotionRole(t)
		source.Schema.execute(
			t,
			connection,
			fmt.Sprintf(
				"ALTER TABLE {added}.wal_records ADD COLUMN switch_%d integer DEFAULT %d",
				switchNumber,
				switchNumber,
			),
		)
		newPoint = source.captureRecoveryPoint(t, chain, fmt.Sprintf("after-full-%d", switchNumber))
		source.restoreRecoveryPoint(t, newPoint)
		for _, target := range gaps {
			source.requireUnreachableTarget(t, target)
		}
	}

	source.setRetainedChains(t, 3)
	var orphanFile string
	// Only the obsolete chain needs WAL that ends before the first retained full. The stream may
	// connect after the obsolete full started, so WAL from before it is not guaranteed, while the
	// incremental's activity always leaves WAL between the two fulls.
	testwait.Until(t, "WAL that only the obsolete chain needs is archived", func(context.Context) error {
		source.Team.App.QuerySQL(
			t,
			&orphanFile,
			"SELECT file_name FROM backups_physical_wal_segments WHERE database_id=? AND timeline_id=? AND end_lsn <= ? AND file_name IS NOT NULL ORDER BY start_lsn LIMIT 1",
			source.Team.Database.Id,
			obsoleteFull.TimelineID,
			firstRetainedFull.StartLSN,
		)
		if orphanFile == "" {
			return errors.New("no archived WAL ends before the first retained full")
		}

		return nil
	})
	require.True(t, source.Team.App.IsLocalStorageFilePresent(t, orphanFile))

	source.Team.App.RunJobPass(t, "backups_physical_cleaner")

	var hasOrphan bool
	source.Team.App.QuerySQL(
		t,
		&hasOrphan,
		"SELECT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=? AND file_name=?)",
		source.Team.Database.Id,
		orphanFile,
	)
	require.False(t, hasOrphan, "physical cleaner must process this database's orphan WAL")

	testwait.Until(t, "storage deletion processes the cleaned orphan WAL", func(context.Context) error {
		source.Team.App.RunJobPass(t, "storage_file_deletion")
		if source.Team.App.IsLocalStorageFilePresent(t, orphanFile) {
			return fmt.Errorf("orphan file %s remains", orphanFile)
		}

		return nil
	})

	source.restoreRecoveryPoint(t, oldPoint)
	source.restoreRecoveryPoint(t, newPoint)
	for _, target := range gaps {
		source.requireUnreachableTarget(t, target)
	}
}

func (source failoverSource) setRetainedChains(t *testing.T, count int) {
	t.Helper()

	response, err := source.Team.Owner.Client.GetBackupConfigWithResponse(t.Context(), source.Team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, "%s", response.Body)
	configuration := response.JSON200
	settings, err := configuration.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	settings.Retention.ChainsCount = count
	require.NoError(t, configuration.Settings.FromPhysicalBackupSettings(settings))
	updated, err := source.Team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		source.Team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: configuration.IsBackupsEnabled,
			Encryption:       configuration.Encryption,
			StorageId:        configuration.StorageId,
			Settings:         configuration.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updated.StatusCode(), "%s", updated.Body)
}
