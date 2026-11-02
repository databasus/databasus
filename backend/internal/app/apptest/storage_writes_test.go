package apptest

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/storages"
)

func TestPublishStorageFile(t *testing.T) {
	t.Parallel()

	t.Run("keeps a file when the deletion job processes expired commit windows", func(t *testing.T) {
		t.Parallel()

		testApp := New(t)
		owner := testApp.CreateAdmin(t)
		workspace := CreateWorkspace(t, owner, "Published files")
		storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)
		fileName := uuid.NewString()
		testApp.PublishStorageFile(
			t,
			storages.StoredFileReference{StorageID: storageID, FileName: fileName},
			strings.NewReader("published"),
		)

		testApp.Clock.Advance(time.Hour)
		testApp.RunJobPass(t, "storage_file_deletion")

		assert.True(t, testApp.IsLocalStorageFilePresent(t, fileName))
	})
}
