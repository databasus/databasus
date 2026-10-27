package backups_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

var errLogicalBackupStillRunning = errors.New("a logical backup of the database is still in progress")

func TestSaveLogicalBackupConfigRacingBackupStart(t *testing.T) {
	t.Run("never leaves a backup on the previous storage of a saved config", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageIDs := []uuid.UUID{tm.createLocalStorage(t), tm.createLocalStorage(t)}
		saveLogicalBackupConfig(t, tm.owner, databaseID, storageIDs[0])

		const roundCount = 8

		for round := range roundCount {
			previousStorageID, nextStorageID := storageIDs[round%2], storageIDs[(round+1)%2]

			saveStatus := make(chan int, 1)
			startSignal := make(chan struct{})

			go func() {
				<-startSignal
				saveStatus <- saveConfigWithStorageFromGoroutine(t, tm.owner, databaseID, nextStorageID)
			}()

			close(startSignal)
			startBackupStatus := startLogicalBackup(t, tm, databaseID)
			roundSaveStatus := <-saveStatus

			require.Equal(t, http.StatusOK, startBackupStatus)

			if roundSaveStatus == http.StatusOK {
				assert.Zero(t, countRows(t, tm,
					"SELECT COUNT(*) FROM logical_backups WHERE database_id = ? AND storage_id = ?",
					databaseID, previousStorageID), "round %d", round)
			} else {
				requireConfigStorage(t, tm, databaseID, previousStorageID)
				assert.Equal(t, http.StatusConflict, roundSaveStatus, "round %d", round)
				storageIDs[0], storageIDs[1] = storageIDs[1], storageIDs[0]
			}

			waitForNoRunningLogicalBackup(t, tm, databaseID)
		}
	})
}

// startLogicalBackup starts a backup through the old route, because no contract operation starts one yet.
func startLogicalBackup(t *testing.T, tm team, databaseID uuid.UUID) int {
	t.Helper()

	encodedBody, err := json.Marshal(map[string]any{"database_id": databaseID})
	require.NoError(t, err)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		tm.app.URL+"/api/v1/backups", bytes.NewReader(encodedBody))
	require.NoError(t, err)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+tm.owner.Token)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	return response.StatusCode
}

func waitForNoRunningLogicalBackup(t *testing.T, tm team, databaseID uuid.UUID) {
	t.Helper()

	testwait.Until(t, "the started logical backup finishes", func(context.Context) error {
		if countRows(t, tm, "SELECT COUNT(*) FROM logical_backups WHERE database_id = ? AND status = 'IN_PROGRESS'",
			databaseID) > 0 {
			return errLogicalBackupStillRunning
		}

		return nil
	})
}
