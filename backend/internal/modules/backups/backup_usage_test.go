package backups_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestLogicalBackupUsage(t *testing.T) {
	t.Parallel()

	for _, engine := range []string{"POSTGRES_LOGICAL", "MYSQL", "MARIADB", "MONGODB"} {
		t.Run("returns the complete catalog total across filters and pages with "+engine, func(t *testing.T) {
			t.Parallel()
			backupTeam := newTeam(t)
			databaseID := insertUsageDatabase(t, backupTeam, engine)
			storageID := backupTeam.createLocalStorage(t)
			otherDatabaseID := insertUsageDatabase(t, backupTeam, engine)
			insertUsageBackup(t, backupTeam, usageBackupSpec{
				DatabaseID: otherDatabaseID,
				StorageID:  storageID,
				Status:     apiclient.LogicalBackupStatusCOMPLETED,
				SizeMb:     750,
			})
			params := newLogicalBackupListParams(databaseID)

			assert.Zero(t, getLogicalBackupPage(t, backupTeam.owner, params).TotalUsageMb)
			assert.InDelta(
				t,
				750.0,
				getLogicalBackupPage(t, backupTeam.owner, newLogicalBackupListParams(otherDatabaseID)).TotalUsageMb,
				0.000001,
			)

			firstBackupID := insertUsageBackup(
				t,
				backupTeam,
				usageBackupSpec{
					DatabaseID: databaseID,
					StorageID:  storageID,
					Status:     apiclient.LogicalBackupStatusCOMPLETED,
					SizeMb:     100,
				},
			)
			insertUsageBackup(
				t,
				backupTeam,
				usageBackupSpec{
					DatabaseID: databaseID,
					StorageID:  storageID,
					Status:     apiclient.LogicalBackupStatusCOMPLETED,
					SizeMb:     50,
				},
			)

			for _, status := range []apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusINPROGRESS, apiclient.LogicalBackupStatusFAILED, apiclient.LogicalBackupStatusCANCELED} {
				insertUsageBackup(
					t,
					backupTeam,
					usageBackupSpec{DatabaseID: databaseID, StorageID: storageID, Status: status, SizeMb: 900},
				)
			}

			params.Status = &[]apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusFAILED}

			filteredPage := getLogicalBackupPage(t, backupTeam.owner, params)

			require.Len(t, filteredPage.Items, 1)
			assert.InDelta(t, 150.0, filteredPage.TotalUsageMb, 0.000001)
			assert.InDelta(t, 900.0, getStorageUsage(t, backupTeam.owner, storageID), 0.000001)

			params.Status = &[]apiclient.LogicalBackupStatus{apiclient.LogicalBackupStatusCOMPLETED}
			params.Limit = new(1)

			firstPage := getLogicalBackupPage(t, backupTeam.owner, params)

			require.NotNil(t, firstPage.NextCursor)
			assert.InDelta(t, 150.0, firstPage.TotalUsageMb, 0.000001)

			params.After = firstPage.NextCursor

			secondPage := getLogicalBackupPage(t, backupTeam.owner, params)

			assert.InDelta(t, firstPage.TotalUsageMb, secondPage.TotalUsageMb, 0.000001)

			deleted, err := backupTeam.owner.Client.DeleteLogicalBackupWithResponse(t.Context(), firstBackupID)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, deleted.StatusCode(), "%s", deleted.Body)
			assert.InDelta(
				t,
				50.0,
				getLogicalBackupPage(t, backupTeam.owner, newLogicalBackupListParams(databaseID)).TotalUsageMb,
				0.000001,
			)
			assert.InDelta(
				t,
				750.0,
				getLogicalBackupPage(t, backupTeam.owner, newLogicalBackupListParams(otherDatabaseID)).TotalUsageMb,
				0.000001,
			)
			assert.InDelta(t, 800.0, getStorageUsage(t, backupTeam.owner, storageID), 0.000001)
			assert.Positive(
				t,
				countRows(
					t,
					backupTeam,
					"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id = ?",
					storageID,
				),
			)
		})
	}
}

func TestGetStorageBackupUsage(t *testing.T) {
	t.Parallel()

	t.Run("returns catalog usage when the saved provider is unreachable", func(t *testing.T) {
		t.Parallel()
		backupTeam := newTeam(t)
		var settings apiclient.StorageSettings
		require.NoError(t, settings.FromS3StorageSettings(apiclient.S3StorageSettings{
			Type: apiclient.S3StorageSettingsTypeS3, S3Bucket: "unreachable-bucket",
			S3Endpoint: new("http://127.0.0.1:1"), S3AccessKey: new("test-key"), S3SecretKey: new("test-secret"),
		}))
		storage := apptest.CreateStorage(t, backupTeam.owner, backupTeam.workspace.Id, settings)
		databaseID := backupTeam.insertLogicalDatabase(t)
		insertUsageBackup(t, backupTeam, usageBackupSpec{
			DatabaseID: databaseID, StorageID: storage.Id, Status: apiclient.LogicalBackupStatusCOMPLETED, SizeMb: 104,
		})

		assert.InDelta(t, 104.0, getStorageUsage(t, backupTeam.owner, storage.Id), 0.000001)
	})

	t.Run("sums eligible records once by their recorded storage without provider reads", func(t *testing.T) {
		t.Parallel()
		backupTeam := newTeam(t)
		databaseID := backupTeam.insertLogicalDatabase(t)
		physicalID := backupTeam.app.InsertPhysicalDatabase(t, backupTeam.workspace.Id)
		storageID, otherStorageID := backupTeam.createLocalStorage(t), backupTeam.createLocalStorage(t)

		assert.Zero(t, getStorageUsage(t, backupTeam.owner, storageID))

		saveLogicalBackupConfig(t, backupTeam.owner, databaseID, otherStorageID)
		insertUsageBackup(
			t,
			backupTeam,
			usageBackupSpec{
				DatabaseID: databaseID,
				StorageID:  storageID,
				Status:     apiclient.LogicalBackupStatusCOMPLETED,
				SizeMb:     100,
			},
		)
		insertUsageBackup(
			t,
			backupTeam,
			usageBackupSpec{
				DatabaseID: databaseID,
				StorageID:  otherStorageID,
				Status:     apiclient.LogicalBackupStatusCOMPLETED,
				SizeMb:     800,
			},
		)
		full := backupTeam.app.InsertPhysicalFullBackup(
			t,
			apptest.PhysicalFullBackupSeed{
				DatabaseID:   physicalID,
				StorageID:    storageID,
				StartLSN:     new("0/0"),
				StopLSN:      new("0/1000000"),
				BackupSizeMb: new(200.0),
			},
		)
		backupTeam.app.InsertPhysicalIncrementalBackup(
			t,
			apptest.PhysicalIncrementalBackupSeed{
				DatabaseID:       physicalID,
				StorageID:        storageID,
				RootFullBackupID: full.ID,
				StartLSN:         new("0/1000000"),
				StopLSN:          new("0/2000000"),
				BackupSizeMb:     new(30.0),
				Status:           "COMPLETED",
			},
		)
		backupTeam.app.InsertPhysicalWalSegment(
			t,
			apptest.PhysicalWalSegmentSeed{
				DatabaseID:       physicalID,
				StorageID:        storageID,
				FileName:         new("archived.zst"),
				WalFilename:      "000000010000000000000001",
				StartLSN:         "0/1000000",
				EndLSN:           "0/2000000",
				CompressedSizeMb: 4,
			},
		)
		backupTeam.app.InsertPhysicalWalSegment(
			t,
			apptest.PhysicalWalSegmentSeed{
				DatabaseID:       physicalID,
				StorageID:        storageID,
				FileName:         nil,
				WalFilename:      "000000010000000000000002",
				StartLSN:         "0/2000000",
				EndLSN:           "0/3000000",
				CompressedSizeMb: 999,
			},
		)
		backupTeam.app.WriteUnpublishedFile(t, storageID, "unpublished")

		assert.InDelta(t, 334.0, getStorageUsage(t, backupTeam.owner, storageID), 0.000001)
		assert.InDelta(t, 800.0, getStorageUsage(t, backupTeam.owner, otherStorageID), 0.000001)
	})

	t.Run("returns usage only to authorized readers with different session roles", func(t *testing.T) {
		t.Parallel()
		backupTeam := newTeam(t)
		storageID := backupTeam.createLocalStorage(t)
		viewer := backupTeam.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		stranger := backupTeam.app.CreateNonAdminUser(t, backupTeam.admin)

		assert.Zero(t, getStorageUsage(t, viewer, storageID))
		assert.Zero(t, getStorageUsage(t, backupTeam.admin, storageID))

		anonymous, err := backupTeam.app.NewClient(t).GetStorageBackupUsageWithResponse(t.Context(), storageID)
		require.NoError(t, err)
		requireProblem(
			t,
			anonymous.StatusCode(),
			anonymous.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		hidden, err := stranger.Client.GetStorageBackupUsageWithResponse(t.Context(), storageID)
		require.NoError(t, err)
		requireProblem(t, hidden.StatusCode(), hidden.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)

		missing, err := stranger.Client.GetStorageBackupUsageWithResponse(t.Context(), uuid.New())
		require.NoError(t, err)
		requireProblem(t, missing.StatusCode(), missing.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		assert.JSONEq(t, string(missing.Body), string(hidden.Body))

		malformedRequest, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			backupTeam.app.URL+"/api/v1/storages/not-a-uuid/backup-usage",
			nil,
		)
		require.NoError(t, err)
		malformedRequest.Header.Set("Authorization", "Bearer "+backupTeam.owner.Token)

		malformedResponse, err := http.DefaultClient.Do(malformedRequest)
		require.NoError(t, err)
		defer func() { _ = malformedResponse.Body.Close() }()

		malformedBody, err := io.ReadAll(malformedResponse.Body)
		require.NoError(t, err)

		requireProblem(
			t,
			malformedResponse.StatusCode,
			malformedBody,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
	})
}

func TestStorageBackupUsageRacingTransfer(t *testing.T) {
	t.Parallel()

	t.Run("checks committed ownership when a transfer holds the storage lock", func(t *testing.T) {
		t.Parallel()
		backupTeam := newTeam(t)
		storageID := backupTeam.createLocalStorage(t)
		viewer := backupTeam.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		target := backupTeam.createWorkspace(t)
		storageLock := backupTeam.app.LockStorageRow(t, storageID)
		type usageRead struct {
			Response *apiclient.GetStorageBackupUsageResponse
			Err      error
		}
		completed := make(chan usageRead, 1)
		go func() {
			response, err := viewer.Client.GetStorageBackupUsageWithResponse(t.Context(), storageID)
			completed <- usageRead{Response: response, Err: err}
		}()
		waitForBlockedConfigRequests(t, backupTeam, 1)

		storageLock.MoveToWorkspace(t, storageID, target.Id)
		storageLock.Commit(t)
		read := <-completed

		require.NoError(t, read.Err)
		requireProblem(
			t,
			read.Response.StatusCode(),
			read.Response.Body,
			http.StatusNotFound,
			apiclient.ProblemCodeNotFound,
		)
	})
}
