package files

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStorageReader(t *testing.T) {
	t.Parallel()

	t.Run("opens the content of a written file", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}

		_, err := files.store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
		require.NoError(t, err)

		storageReader, err := files.store.GetStorageReader(t.Context(), reference.StorageID)
		require.NoError(t, err)

		openedFile, err := storageReader.OpenFile(t.Context(), reference.FileName)
		require.NoError(t, err)
		t.Cleanup(func() { _ = openedFile.Close() })

		readContent, err := io.ReadAll(openedFile)
		require.NoError(t, err)
		assert.Equal(t, "payload", string(readContent))
	})

	t.Run("fails when the storage cannot be located", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.locator.err = errors.New("no such storage")

		_, err := files.store.GetStorageReader(t.Context(), files.createStorage(t))

		assert.ErrorIs(t, err, files.locator.err)
	})
}
