package backuping_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func (team *backupTeam) useUnreachableStorage(t *testing.T) {
	t.Helper()
	var settings apiclient.StorageSettings
	require.NoError(
		t,
		settings.FromS3StorageSettings(
			apiclient.S3StorageSettings{
				Type:        apiclient.S3StorageSettingsTypeS3,
				S3Bucket:    "backups",
				S3Region:    new("us-east-1"),
				S3Endpoint:  new("http://127.0.0.1:1"),
				S3AccessKey: new("test-access"),
				S3SecretKey: new("test-secret"),
			},
		),
	)

	response, err := team.Owner.Client.CreateStorageWithResponse(
		t.Context(),
		apiclient.StorageCreate{Name: "Unavailable S3", WorkspaceId: team.Database.WorkspaceId, Settings: settings},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), "%s", response.Body)

	team.StorageID = response.JSON201.Id
	apptest.SaveBackupConfigWithStorage(
		t,
		team.Owner,
		apptest.StorageUse{DatabaseID: team.Database.Id, StorageID: team.StorageID},
	)
}

func TestPhysicalBackupStorageFailure(t *testing.T) {
	t.Parallel()

	t.Run("reports storage upload failure when the S3 endpoint cannot be reached", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		team.useUnreachableStorage(t)

		backup := team.runBackup(t, "full")
		require.Equal(t, "ERROR", backup.Status)

		team.assertTerminalDetails(t, backup, "STORAGE_UPLOAD_FAILED")
		team.assertNoRunningBackup(t)
		assert.NotNil(t, backup.FailMessage)
	})
}
