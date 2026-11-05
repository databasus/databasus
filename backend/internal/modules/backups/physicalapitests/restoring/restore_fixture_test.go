package restoring_test

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/storages"
)

type (
	restoreTokenResponse = apiclient.PhysicalRestoreToken
)

func createStreamableFullBackup(t *testing.T, catalog *catalogFixture) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	return catalog.insertStreamableFull(t, map[string]string{"PG_VERSION": "17\n", "base/1/1259": "heap-bytes"})
}

func (catalog *catalogFixture) insertStreamableFull(
	t *testing.T,
	files map[string]string,
) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StopLSN:    fixtureLSN(segmentBytes),
		},
	)
	full.ManifestFileName = new(*full.FileName + ".manifest")

	blob := zstdTar(t, files)
	full.BackupSizeMb = new(float64(len(blob)) / (1024 * 1024))

	catalog.App.PublishStorageFile(
		t,
		storages.StoredFileReference{StorageID: catalog.StorageID, FileName: *full.FileName},
		bytes.NewReader(blob),
	)
	catalog.App.PublishStorageFile(
		t,
		storages.StoredFileReference{StorageID: catalog.StorageID, FileName: *full.ManifestFileName},
		bytes.NewReader([]byte(`{"PostgreSQL-Backup-Manifest-Version":2}`)),
	)
	return catalog.insertFull(t, full)
}

func (catalog *catalogFixture) holdRestoreStream(t *testing.T) {
	t.Helper()
	largeHeap := make([]byte, 32*1024*1024)
	_, err := rand.Read(largeHeap)
	require.NoError(t, err)

	full := catalog.insertStreamableFull(t, map[string]string{"PG_VERSION": "17\n", "base/1/1259": string(largeHeap)})
	var token restoreTokenResponse
	catalog.call(
		t,
		catalogCall{
			Request:       apptest.PhysicalBackupTokenRequest(full.ID),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusCreated,
			Destination:   &token,
		},
	)

	response := catalog.App.OpenPhysicalAPI(
		t, apptest.Session{},
		apptest.PhysicalDownloadRequest(token.Token),
	)
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func zstdTar(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var rawTar bytes.Buffer
	tarWriter := tar.NewWriter(&rawTar)
	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Name: name, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg,
		}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(rawTar.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	return compressed.Bytes()
}

func readTarEntryNames(t *testing.T, body []byte) []string {
	t.Helper()

	var names []string
	tarReader := tar.NewReader(bytes.NewReader(body))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, header.Name)
	}

	return names
}
