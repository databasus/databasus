package storages_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const rcloneConfig = "[myremote]\ntype = s3\nprovider = AWS\naccess_key_id = test\nsecret_access_key = secret\n"

func TestRcloneStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns 403 permission_denied and creates nothing when a member creates an rclone storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)

			response, err := tm.owner.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Remote",
				Settings:    newRcloneSettings(t, apiclient.RcloneStorageSettings{ConfigContent: new(rcloneConfig)}),
			})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
		})

	t.Run("lists the rclone storage without its configuration when an instance admin creates it",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)

			createdStorage := createStorage(t, tm.admin, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Remote",
				Settings: newRcloneSettings(t, apiclient.RcloneStorageSettings{
					ConfigContent: new(rcloneConfig),
					RemotePath:    new("/backups"),
				}),
			})

			listedStorages := listStorages(t, tm.owner, tm.workspace.Id)
			require.Len(t, listedStorages, 1)
			assert.Equal(t, createdStorage.Id, listedStorages[0].Id)

			listedRclone, err := listedStorages[0].Settings.AsRcloneStorageSettings()
			require.NoError(t, err)
			assert.Nil(t, listedRclone.ConfigContent)
			assert.Equal(t, new("/backups"), listedRclone.RemotePath)
		})

	t.Run("returns 403 permission_denied and keeps the name when a member renames an rclone storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			rcloneStorage := createStorage(t, tm.admin, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Remote",
				Settings: newRcloneSettings(t, apiclient.RcloneStorageSettings{
					ConfigContent: new(rcloneConfig),
					RemotePath:    new("/backups"),
				}),
			})

			response := updateStorage(t, tm.owner, rcloneStorage.Id, apiclient.StorageUpdate{
				Name:     "Renamed remote",
				Settings: newRcloneSettings(t, apiclient.RcloneStorageSettings{RemotePath: new("/backups")}),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Equal(t, "Remote", getStorage(t, tm.owner, rcloneStorage.Id).Name)
		})

	t.Run("returns 403 permission_denied and keeps the S3 storage when a member sends rclone settings to it",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			s3Storage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Offsite",
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:    "backups",
					S3AccessKey: new("access"),
					S3SecretKey: new("secret"),
				}),
			})

			response := updateStorage(t, tm.owner, s3Storage.Id, apiclient.StorageUpdate{
				Name:     "Offsite",
				Settings: newRcloneSettings(t, apiclient.RcloneStorageSettings{ConfigContent: new(rcloneConfig)}),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			_, err := getStorage(t, tm.owner, s3Storage.Id).Settings.AsS3StorageSettings()
			require.NoError(t, err)
		})

	t.Run("returns 403 permission_denied when a member tests unsaved rclone settings", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
			WorkspaceId: tm.workspace.Id,
			StorageId:   nil,
			Settings:    newRcloneSettings(t, apiclient.RcloneStorageSettings{ConfigContent: new(rcloneConfig)}),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})
}
