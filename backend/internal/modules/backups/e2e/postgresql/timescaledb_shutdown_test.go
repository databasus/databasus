package postgresql

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // The subtests share one source server.
func TestTimescaleRestoreShutdown(t *testing.T) {
	t.Parallel()
	endpoint := containers.StartTimescaleDB(t, timescaledbImage)
	for _, scenario := range []struct {
		name                     string
		isCleanupStalled         bool
		isCanceledBeforeShutdown bool
	}{
		{name: "leaves restoring mode after user cancellation", isCleanupStalled: false, isCanceledBeforeShutdown: true},
		{name: "leaves restoring mode when shutdown interrupts the archive", isCleanupStalled: false, isCanceledBeforeShutdown: false},
		{name: "records shutdown failure when cleanup exceeds the shutdown cutoff", isCleanupStalled: true, isCanceledBeforeShutdown: false},
		{name: "keeps cancellation when shutdown shortens an existing cleanup", isCleanupStalled: true, isCanceledBeforeShutdown: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := startBlockedTimescaleRestore(t, endpoint)
			fixture.proxy.isHolding.Store(scenario.isCleanupStalled)

			if scenario.isCanceledBeforeShutdown {
				cancelResponse, err := fixture.client.CancelRestoreWithResponse(t.Context(), fixture.restoreID)
				require.NoError(t, err)
				require.Equal(t, http.StatusAccepted, cancelResponse.StatusCode())
				if scenario.isCleanupStalled {
					testwait.Until(t, "the cleanup connection is stalled", func(context.Context) error {
						if fixture.proxy.heldCount.Load() == 0 {
							return fmt.Errorf("cleanup has not connected")
						}
						return nil
					})
				} else {
					testwait.Until(t, "the canceled restore finishes cleanup", func(ctx context.Context) error {
						canceled, err := fixture.client.GetRestoreWithResponse(ctx, fixture.restoreID)
						if err != nil {
							return err
						}
						if canceled.JSON200.Status != apiclient.RestoreStatusCANCELED {
							return fmt.Errorf("restore is still %s", canceled.JSON200.Status)
						}
						return nil
					})
					require.False(t, isTimescaleTargetRestoring(t, fixture.server, fixture.targetDatabaseName))
				}
			}

			fixture.process.Stop(t)
			restoreResponse, err := fixture.team.Owner.Client.GetRestoreWithResponse(t.Context(), fixture.restoreID)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, restoreResponse.StatusCode())

			if scenario.isCanceledBeforeShutdown {
				assert.Equal(t, apiclient.RestoreStatusCANCELED, restoreResponse.JSON200.Status)
			} else {
				assert.Equal(t, apiclient.RestoreStatusFAILED, restoreResponse.JSON200.Status)
				require.NotNil(t, restoreResponse.JSON200.FailMessage)
				assert.Contains(t, *restoreResponse.JSON200.FailMessage, "restore interrupted by application shutdown")
			}

			assert.Equal(
				t,
				scenario.isCleanupStalled,
				isTimescaleTargetRestoring(t, fixture.server, fixture.targetDatabaseName),
			)

			if scenario.isCleanupStalled {
				log, err := os.ReadFile(fixture.process.outputPath)
				require.NoError(t, err)
				assert.Contains(t, string(log), "target database may be left in restoring mode")
			}

			fixture.proxy.Release()
		})
	}
}

func isTimescaleTargetRestoring(t *testing.T, server postgresqlServer, databaseName string) bool {
	t.Helper()
	connection := server.open(t, databaseName)
	var restoringMode string
	require.NoError(t, connection.GetContext(t.Context(), &restoringMode, "SHOW timescaledb.restoring"))
	require.NoError(t, connection.Close())
	return restoringMode == "on"
}

func TestTimescaleRestoreForcedShutdown(t *testing.T) {
	t.Parallel()
	t.Run("recovers the outcome on restart when archive input prevents draining", func(t *testing.T) {
		t.Parallel()

		endpoint := containers.StartTimescaleDB(t, timescaledbImage)
		server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, server.newSettings())
		backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)

		var backupFileName string
		tm.App.QuerySQL(t, &backupFileName, "SELECT file_name FROM backups_logical WHERE id = ?", backup.Id)
		backupPath := tm.App.GetLocalStorageFilePath(backupFileName)
		require.NoError(t, os.Remove(backupPath))
		require.NoError(t, syscall.Mkfifo(backupPath, 0o600))

		// A real FIFO keeps the storage read blocked even after the restore context ends.
		archive, err := os.OpenFile(backupPath, os.O_RDWR, 0o600)
		require.NoError(t, err)
		defer func() { _ = archive.Close() }()

		targetName, _ := server.createRestoreDatabase(t, "forced_cleanup_target")
		process := startTimescaleServer(t, tm.App.GetServerConfig())
		client := apiclient.New(t, process.URL, apiclient.WithSessionToken(tm.Owner.Token))

		response, err := client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
			BackupId: backup.Id, Target: toPostgresqlRestoreTarget(t, server.newTarget(targetName)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))

		testwait.Until(t, "pre-restore completes before the blocked archive read", func(context.Context) error {
			if !isTimescaleTargetRestoring(t, server, targetName) {
				return fmt.Errorf("restoring mode is not enabled")
			}
			return nil
		})

		process.Stop(t)
		assert.True(t, isTimescaleTargetRestoring(t, server, targetName))
		log, err := os.ReadFile(process.outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(log), "shutdown steps did not finish within the drain limit")
		assert.Contains(t, string(log), "backup runs")

		unfinished, err := tm.Owner.Client.GetRestoreWithResponse(t.Context(), response.JSON202.Id)
		require.NoError(t, err)
		require.Equal(t, apiclient.RestoreStatusINPROGRESS, unfinished.JSON200.Status)

		restarted := startTimescaleServer(t, tm.App.GetServerConfig())
		recovered, err := tm.Owner.Client.GetRestoreWithResponse(t.Context(), response.JSON202.Id)
		require.NoError(t, err)
		require.Equal(t, apiclient.RestoreStatusFAILED, recovered.JSON200.Status)
		require.NotNil(t, recovered.JSON200.FailMessage)
		assert.Contains(t, *recovered.JSON200.FailMessage, "Restore failed due to application restart")
		restarted.Stop(t)
	})
}
