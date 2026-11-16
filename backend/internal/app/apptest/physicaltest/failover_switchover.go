package physicaltest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

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
	var switchPoints []recoveryPoint
	fullCount := len(source.Team.listBackups(t, "FULL"))

	for switchNumber := 1; switchNumber <= 2; switchNumber++ {
		previous := source.Cluster.GetIdentity(t)
		promoted := source.Cluster.Switchover(t)
		require.NotEqual(t, previous.Node, promoted.Node)
		require.Greater(t, promoted.Timeline, previous.Timeline)
		connection = source.connect(t)
		source.Team.waitForTimelineHistory(t, promoted.Timeline)
		source.Team.requirePublishedTimelineHistory(t, connection)
		source.Team.requireRestrictedPromotionRole(t)

		// A target right after the switch restores from the chain across the fork point before any
		// backup runs on the promoted timeline.
		switchPoint := source.markReplayablePoint(t, connection, fmt.Sprintf("after-switch-%d", switchNumber))
		source.restoreRecoveryPoint(t, switchPoint)
		switchPoints = append(switchPoints, switchPoint)

		source.Schema.execute(
			t,
			connection,
			fmt.Sprintf(
				"ALTER TABLE {added}.wal_records ADD COLUMN switch_%d integer DEFAULT %d",
				switchNumber,
				switchNumber,
			),
		)
		chain = source.Team.buildIncremental(
			t,
			connection,
			incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1 + switchNumber},
		)
		require.Len(
			t,
			source.Team.listBackups(t, "FULL"),
			fullCount,
			"the INCREMENTAL extends the chain across the switch",
		)
		require.Equal(t, promoted.Timeline, chain[len(chain)-1].TimelineID)
		newPoint = source.captureRecoveryPoint(t, chain, fmt.Sprintf("after-incremental-%d", switchNumber))
		source.restoreRecoveryPoint(t, newPoint)
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
	for _, switchPoint := range switchPoints {
		source.restoreRecoveryPoint(t, switchPoint)
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
