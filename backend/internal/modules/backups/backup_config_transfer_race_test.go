package backups_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestSaveBackupConfigRacingDatabaseTransfer(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"logical", "physical"} {
		for _, scenario := range []string{"former storage", "destination storage", "former member"} {
			t.Run(
				"uses the committed workspace after a transfer wins against a "+method+" save with "+scenario,
				func(t *testing.T) {
					t.Parallel()
					tm := newTeam(t)
					databaseID := tm.insertLogicalDatabase(t)
					if method == "physical" {
						databaseID = tm.app.InsertPhysicalDatabase(t, tm.workspace.Id)
					}
					sourceStorageID := tm.createLocalStorage(t)
					formerStorageID := tm.createLocalStorage(t)

					backupConfig := apptest.GetBackupConfig(t, tm.owner, databaseID)
					configInput := apiclient.BackupConfigInput{
						IsBackupsEnabled: false, StorageId: &sourceStorageID,
						Encryption: backupConfig.Encryption, Settings: backupConfig.Settings,
					}
					initial := updateBackupConfig(t, tm.owner, databaseID, configInput)
					require.Equal(t, http.StatusOK, initial.StatusCode(), "%s", initial.Body)

					destination := tm.createWorkspace(t)
					transferStorageID := tm.app.CreateLocalStorage(t, tm.owner, destination.Id)
					destinationStorageID := tm.app.CreateLocalStorage(t, tm.owner, destination.Id)
					actor := tm.owner
					if scenario == "former member" {
						actor = tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
					}

					configInput.StorageId = &destinationStorageID
					if scenario == "former storage" || scenario == "former member" {
						configInput.StorageId = &formerStorageID
					}

					storageLock := tm.app.LockStorageRow(t, transferStorageID)
					transferInput := newDatabaseTransfer(
						t,
						destination.Id,
						newStorageSwitchedOnTransfer(t, transferStorageID),
					)

					transferred := make(chan struct {
						Response *apiclient.TransferDatabaseResponse
						Err      error
					}, 1)
					go func() {
						response, err := tm.owner.Client.TransferDatabaseWithResponse(
							t.Context(),
							databaseID,
							transferInput,
						)
						transferred <- struct {
							Response *apiclient.TransferDatabaseResponse
							Err      error
						}{Response: response, Err: err}
					}()

					waitForBlockedConfigRequests(t, tm, 1)

					saved := make(chan struct {
						Response *apiclient.UpdateBackupConfigResponse
						Err      error
					}, 1)
					go func() {
						response, err := actor.Client.UpdateBackupConfigWithResponse(
							t.Context(),
							databaseID,
							configInput,
						)
						saved <- struct {
							Response *apiclient.UpdateBackupConfigResponse
							Err      error
						}{Response: response, Err: err}
					}()

					waitForBlockedConfigRequests(t, tm, 2)
					storageLock.Commit(t)
					transferOutcome, saveOutcome := <-transferred, <-saved

					require.NoError(t, transferOutcome.Err)
					require.Equal(
						t,
						http.StatusNoContent,
						transferOutcome.Response.StatusCode(),
						"%s",
						transferOutcome.Response.Body,
					)
					require.NoError(t, saveOutcome.Err)

					if scenario == "destination storage" {
						require.Equal(
							t,
							http.StatusOK,
							saveOutcome.Response.StatusCode(),
							"%s",
							saveOutcome.Response.Body,
						)
						require.NotNil(t, saveOutcome.Response.JSON200)
						assert.Equal(t, &destinationStorageID, saveOutcome.Response.JSON200.StorageId)
						assert.Equal(
							t,
							&destinationStorageID,
							apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId,
						)
					} else {
						status := http.StatusBadRequest
						if scenario == "former member" {
							status = http.StatusNotFound
						}
						assert.Equal(t, status, saveOutcome.Response.StatusCode(), "%s", saveOutcome.Response.Body)
						assert.Equal(t, &transferStorageID, apptest.GetBackupConfig(t, tm.owner, databaseID).StorageId)
					}

					assert.Zero(t, countRows(
						t,
						tm,
						"SELECT count(*) FROM storages_pending_file_deletions WHERE cleanup_database_id=? AND cleanup_workspace_id=?",
						databaseID,
						tm.workspace.Id,
					))
				},
			)
		}
	}
}

func waitForBlockedConfigRequests(t *testing.T, tm team, minimum int) {
	t.Helper()
	testwait.Until(t, "the config requests wait on transaction locks", func(context.Context) error {
		var blocked int
		tm.app.QuerySQL(
			t,
			&blocked,
			"SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND cardinality(pg_blocking_pids(pid))>0",
		)
		if blocked < minimum {
			return fmt.Errorf("only %d requests are blocked", blocked)
		}
		return nil
	})
}
