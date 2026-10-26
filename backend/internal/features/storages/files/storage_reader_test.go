package storage_files

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetStorageReader_WhenFileWasWritten_OpensItsContent(t *testing.T) {
	store := newTestStore(&fakeLocator{provider: &fakeProvider{}})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
	require.NoError(t, err)

	storageReader, err := store.GetStorageReader(t.Context(), reference.StorageID)
	require.NoError(t, err)

	openedFile, err := storageReader.OpenFile(t.Context(), reference.FileName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = openedFile.Close() })

	readContent, err := io.ReadAll(openedFile)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(readContent))
}

func Test_GetStorageReader_WhenStorageCannotBeLocated_Fails(t *testing.T) {
	store := newTestStore(&fakeLocator{provider: &fakeProvider{}, err: assert.AnError})

	_, err := store.GetStorageReader(t.Context(), createStorageRow(t))

	assert.ErrorIs(t, err, assert.AnError)
}
