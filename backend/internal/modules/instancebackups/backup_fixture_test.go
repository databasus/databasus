package instancebackups_test

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const (
	productionJob = "instancebackups_backup_production"
	deliveryJob   = "instancebackups_copy_delivery"
	retentionJob  = "instancebackups_retention"
)

// backUp runs the dump and the delivery of the backup a destination or the schedule asked for.
func (o instanceOwner) backUp(t *testing.T) apiclient.InstanceBackup {
	t.Helper()

	o.App.RunJobPass(t, productionJob)
	o.App.RunJobPass(t, deliveryJob)

	return o.getLatestBackup(t)
}

func (o instanceOwner) startBackup(t *testing.T) apiclient.InstanceBackup {
	t.Helper()

	response, err := o.RootAdmin.Client.StartInstanceBackupWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, response.JSON202, string(response.Body))

	return *response.JSON202
}

func (o instanceOwner) listBackups(t *testing.T) []apiclient.InstanceBackup {
	t.Helper()

	limit := 100

	response, err := o.RootAdmin.Client.ListInstanceBackupsWithResponse(t.Context(),
		&apiclient.ListInstanceBackupsParams{Limit: &limit, After: nil})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return response.JSON200.Items
}

func (o instanceOwner) getLatestBackup(t *testing.T) apiclient.InstanceBackup {
	t.Helper()

	backups := o.listBackups(t)
	require.NotEmpty(t, backups)

	return backups[0]
}

func (o instanceOwner) getBackup(t *testing.T, backupID uuid.UUID) apiclient.InstanceBackup {
	t.Helper()

	response, err := o.RootAdmin.Client.GetInstanceBackupWithResponse(t.Context(), backupID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func getCopy(t *testing.T, backup apiclient.InstanceBackup, storageID uuid.UUID) apiclient.InstanceBackupCopy {
	t.Helper()

	for _, backupCopy := range backup.Copies {
		if backupCopy.StorageId != nil && *backupCopy.StorageId == storageID {
			return backupCopy
		}
	}

	t.Fatalf("the backup has no copy in the storage %s", storageID)

	return apiclient.InstanceBackupCopy{}
}

func (o instanceOwner) createDownloadToken(
	t *testing.T,
	backupID uuid.UUID,
	copyID uuid.UUID,
) apiclient.InstanceBackupDownloadToken {
	t.Helper()

	response, err := o.RootAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), backupID,
		apiclient.InstanceBackupDownloadTokenCreate{CopyId: copyID})
	require.NoError(t, err)
	require.NotNil(t, response.JSON201, string(response.Body))

	return *response.JSON201
}

func (o instanceOwner) getSpoolFileNames(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(o.App.GetServerConfig().InstanceBackupSpoolFolder)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}
	}

	require.NoError(t, err)

	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		fileNames = append(fileNames, entry.Name())
	}

	return fileNames
}

// runDeliveryInBackground starts a delivery pass whose context the test ends, as a shutdown does.
func (o instanceOwner) runDeliveryInBackground(ctx context.Context, t *testing.T) <-chan error {
	t.Helper()

	passResult := make(chan error, 1)

	go func() { passResult <- o.App.RunJobPassWithContext(ctx, deliveryJob) }()

	return passResult
}
