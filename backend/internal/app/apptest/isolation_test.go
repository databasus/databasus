package apptest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages"
)

func TestApplicationIsolation(t *testing.T) {
	t.Parallel()

	t.Run("preserves data and files when a neighbouring application shuts down", func(t *testing.T) {
		t.Parallel()

		survivor := New(t)
		survivorOwner := survivor.SignUp(t, "same@example.com")
		survivorWorkspace := CreateWorkspace(t, survivorOwner, "Survivor")
		survivorStorageID := survivor.CreateLocalStorage(t, survivorOwner, survivorWorkspace.Id)
		survivorReference := storages.StoredFileReference{StorageID: survivorStorageID, FileName: "same.backup"}
		require.NoError(
			t,
			survivor.app.PublishStorageFile(t.Context(), survivorReference, bytes.NewBufferString("survivor")),
		)

		streamReader, streamWriter := io.Pipe()
		t.Cleanup(func() { _ = streamReader.Close(); _ = streamWriter.Close() })
		writeOutcome := make(chan error, 1)
		go func() {
			writeOutcome <- survivor.app.PublishStorageFile(t.Context(), storages.StoredFileReference{StorageID: survivorStorageID, FileName: "inflight.backup"}, streamReader)
		}()
		_, err := streamWriter.Write([]byte("started "))
		require.NoError(t, err)
		testwait.Until(t, "the survivor stages its active write", func(context.Context) error {
			if !survivor.IsLocalStorageStagingFilePresent(t, "inflight.backup") {
				return fmt.Errorf("write has not staged its file")
			}
			return nil
		})

		t.Run("keeps both applications independent while alive", func(t *testing.T) {
			neighbour := New(t)
			neighbourOwner := neighbour.SignUp(t, "same@example.com")
			neighbourWorkspace := CreateWorkspace(t, neighbourOwner, "Neighbour")
			neighbourStorageID := neighbour.CreateLocalStorage(t, neighbourOwner, neighbourWorkspace.Id)
			require.NoError(
				t,
				neighbour.app.PublishStorageFile(
					t.Context(),
					storages.StoredFileReference{StorageID: neighbourStorageID, FileName: "same.backup"},
					bytes.NewBufferString("neighbour"),
				),
			)

			assert.False(t, neighbour.IsLocalStorageStagingFilePresent(t, "inflight.backup"))
			assert.NotEqual(t, survivor.databaseDsn, neighbour.databaseDsn)
			assert.NotEqual(t, survivor.tempFolder, neighbour.tempFolder)
			assert.NotEqual(t, survivor.dataFolder, neighbour.dataFolder)
			assert.NotEqual(
				t,
				filepath.Join(survivor.dataFolder, "wal-queue"),
				filepath.Join(neighbour.dataFolder, "wal-queue"),
			)
			assert.NotEqual(t, survivor.secretKeyPath, neighbour.secretKeyPath)

			survivorKey, err := os.ReadFile(survivor.secretKeyPath)
			require.NoError(t, err)
			neighbourKey, err := os.ReadFile(neighbour.secretKeyPath)
			require.NoError(t, err)
			assert.NotEqual(t, survivorKey, neighbourKey)
			assertLocalContent(t, survivor, "survivor")
			assertLocalContent(t, neighbour, "neighbour")
			assertWorkspaceNames(t, survivorOwner, "Survivor")
			assertWorkspaceNames(t, neighbourOwner, "Neighbour")
		})

		_, err = streamWriter.Write([]byte("finished"))
		require.NoError(t, err)
		require.NoError(t, streamWriter.Close())
		require.NoError(t, <-writeOutcome)

		streamedContent, err := os.ReadFile(survivor.GetLocalStorageFilePath("inflight.backup"))
		require.NoError(t, err)
		assert.Equal(t, "started finished", string(streamedContent))

		assertWorkspaceNames(t, survivorOwner, "Survivor")
		assertLocalContent(t, survivor, "survivor")
		survivorReference.FileName = "after-shutdown.backup"
		require.NoError(
			t,
			survivor.app.PublishStorageFile(t.Context(), survivorReference, bytes.NewBufferString("after shutdown")),
		)
		storedContent, err := os.ReadFile(filepath.Join(survivor.dataFolder, survivorReference.FileName))
		require.NoError(t, err)
		assert.Equal(t, "after shutdown", string(storedContent))
	})
}

func assertLocalContent(t *testing.T, application *App, content string) {
	t.Helper()
	storedContent, err := os.ReadFile(filepath.Join(application.dataFolder, "same.backup"))
	require.NoError(t, err)
	assert.Equal(t, content, string(storedContent))
}

func assertWorkspaceNames(t *testing.T, owner Session, name string) {
	t.Helper()
	response, err := owner.Client.ListWorkspacesWithResponse(t.Context(), &apiclient.ListWorkspacesParams{})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
	require.Len(t, response.JSON200.Items, 1)
	assert.Equal(t, name, response.JSON200.Items[0].Name)
}
