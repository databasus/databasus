package walstreaming_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func (team *walTeam) useUnavailableStorage(t *testing.T) {
	t.Helper()
	var settings apiclient.StorageSettings
	require.NoError(
		t,
		settings.FromS3StorageSettings(
			apiclient.S3StorageSettings{
				Type:        apiclient.S3StorageSettingsTypeS3,
				S3Bucket:    "backups",
				S3Region:    new(containers.MinioRegion),
				S3Endpoint:  new("http://127.0.0.1:1"),
				S3AccessKey: new(containers.MinioRootUser),
				S3SecretKey: new(containers.MinioRootPassword),
			},
		),
	)

	response, err := team.Owner.Client.CreateStorageWithResponse(
		t.Context(),
		apiclient.StorageCreate{Name: "Unavailable WAL S3", WorkspaceId: team.Workspace.Id, Settings: settings},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), "%s", response.Body)
	team.StorageID = response.JSON201.Id
}

func (team *walTeam) recoverStorage(t *testing.T) *minio.Client {
	t.Helper()
	endpoint := containers.StartMinio(t)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)

	client, err := minio.New(
		address,
		&minio.Options{
			Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
			Region: containers.MinioRegion,
			Secure: false,
		},
	)
	require.NoError(t, err)
	require.NoError(
		t,
		client.MakeBucket(t.Context(), "backups", minio.MakeBucketOptions{Region: containers.MinioRegion}),
	)

	var settings apiclient.StorageSettings
	require.NoError(
		t,
		settings.FromS3StorageSettings(
			apiclient.S3StorageSettings{
				Type:        apiclient.S3StorageSettingsTypeS3,
				S3Bucket:    "backups",
				S3Region:    new(containers.MinioRegion),
				S3Endpoint:  new("http://" + address),
				S3AccessKey: new(containers.MinioRootUser),
				S3SecretKey: new(containers.MinioRootPassword),
			},
		),
	)

	response, err := team.Owner.Client.UpdateStorageWithResponse(
		t.Context(),
		team.StorageID,
		apiclient.StorageUpdate{Name: "Recovered WAL S3", Settings: settings},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	return client
}

func TestRecoverQueuedPhysicalWal(t *testing.T) {
	t.Parallel()

	t.Run("retains local segments after upload failure and commits them after S3 recovers", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.useUnavailableStorage(t)
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)

		var claimedSegments []struct {
			ID          uuid.UUID
			WalFilename string
		}
		testwait.Until(t, "the failed upload leaves WAL in the local queue", func(context.Context) error {
			reading := team.live(t)
			if reading.Current == nil || reading.Current.LocalQueueBytes == 0 {
				return fmt.Errorf("queue is empty")
			}
			var committed int
			team.App.QuerySQL(
				t,
				&committed,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
				team.Database.Id,
			)
			if committed != 0 {
				return fmt.Errorf("unreachable storage unexpectedly published WAL")
			}
			team.App.QuerySQL(
				t,
				&claimedSegments,
				"SELECT id, wal_filename FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NULL LIMIT 1",
				team.Database.Id,
			)
			if len(claimedSegments) == 0 {
				return fmt.Errorf("the uploader has not claimed a queued segment")
			}
			return nil
		})
		testwait.Until(
			t,
			"a failed WAL write is discarded and its catalog claim is released",
			func(context.Context) error {
				var releasedFailures int
				team.App.QuerySQL(
					t,
					&releasedFailures,
					`SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name LIKE ? AND generation>0 AND NOT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE id=?)`,
					team.StorageID,
					team.Database.Id.String()+"-WAL-tl%-"+claimedSegments[0].WalFilename+"-%.zst",
					claimedSegments[0].ID,
				)
				if releasedFailures == 0 {
					return fmt.Errorf("no failed WAL write has released its claim")
				}
				return nil
			},
		)

		assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))
		client := team.recoverStorage(t)
		team.waitForWal(t, 1)

		var files []string
		team.App.QuerySQL(
			t,
			&files,
			"SELECT file_name FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
			team.Database.Id,
		)
		require.NotEmpty(t, files)
		for _, file := range files {
			object, err := client.StatObject(t.Context(), "backups", file, minio.StatObjectOptions{})
			require.NoError(t, err)
			assert.Positive(t, object.Size)
			sidecar, err := client.StatObject(t.Context(), "backups", file+".metadata", minio.StatObjectOptions{})
			require.NoError(t, err)
			assert.Positive(t, sidecar.Size)
		}

		testwait.Until(t, "successful uploads drain the staging queue", func(context.Context) error {
			reading := team.live(t)
			if reading.Current == nil || reading.Current.LocalQueueBytes != 0 {
				return fmt.Errorf("queue has not drained")
			}
			return nil
		})
	})
}
