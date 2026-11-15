package backups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestDownloadFromRecoveredLocalStorage(t *testing.T) {
	t.Parallel()

	t.Run("keeps a historical backup missing on the new host unavailable after validation", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		databaseID := tm.insertLogicalDatabase(t)
		storageID := tm.createLocalStorage(t)
		backupID := insertLogicalBackupRecord(t, tm, databaseID, storageID,
			apiclient.LogicalBackupStatusCOMPLETED, tm.app.Clock.Now().Add(-time.Hour))
		tm.app.MarkStorageRecovered(t, storageID)

		unvalidatedDownload := requestBackupDownload(t, tm, backupID, issueDownloadToken(t, tm.owner, backupID).Token)
		validation, err := tm.owner.Client.TestStorageWithResponse(t.Context(), storageID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, validation.StatusCode(), string(validation.Body))
		validatedDownload := requestBackupDownload(t, tm, backupID, issueDownloadToken(t, tm.owner, backupID).Token)

		requireProblem(t, unvalidatedDownload.StatusCode(), unvalidatedDownload.Body,
			http.StatusConflict, apiclient.ProblemCodeStorageUnavailable)
		require.NotNil(t, validation.JSON200.RecoveryValidationStatus)
		assert.Equal(
			t,
			apiclient.StorageRecoveryValidationStatusAVAILABLE,
			*validation.JSON200.RecoveryValidationStatus,
		)
		requireProblem(t, validatedDownload.StatusCode(), validatedDownload.Body,
			http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}
