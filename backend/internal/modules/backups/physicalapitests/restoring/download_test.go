package restoring_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestDownloadPhysicalRestoreSet(t *testing.T) {
	t.Parallel()

	t.Run("does not compress the restore stream when the client accepts gzip", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := createStreamableFullBackup(t, catalog)
		var token restoreTokenResponse
		catalog.call(t, catalogCall{
			Request:       apptest.PhysicalBackupTokenRequest(full.ID),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusCreated,
			Destination:   &token,
		})

		response, err := catalog.App.NewClient(t).DownloadPhysicalRestoreSetWithResponse(
			t.Context(), &apiclient.DownloadPhysicalRestoreSetParams{Token: token.Token},
			func(_ context.Context, request *http.Request) error {
				request.Header.Set("Accept-Encoding", "gzip")
				return nil
			},
		)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, response.StatusCode())
		assert.False(t, response.HTTPResponse.Uncompressed)
		require.Empty(t, response.HTTPResponse.Header.Get("Content-Encoding"))
		assert.Equal(t, "application/x-tar", response.HTTPResponse.Header.Get("Content-Type"))
		assert.Contains(
			t,
			response.HTTPResponse.Header.Get("Content-Disposition"),
			"restore-"+catalog.DatabaseID.String()+".tar",
		)

		entries := readTarEntries(t, response.Body)
		require.Contains(t, entries, "full/base.tar.zst")
		assert.Equal(t, `{"PostgreSQL-Backup-Manifest-Version":2}`, entries["full/backup_manifest"])
		assert.Contains(t, entries["MANIFEST.sha256"], sha256Hex(entries["full/base.tar.zst"])+"  full/base.tar.zst")

		decoder, err := zstd.NewReader(nil)
		require.NoError(t, err)
		defer decoder.Close()
		backupArchive, err := decoder.DecodeAll([]byte(entries["full/base.tar.zst"]), nil)
		require.NoError(t, err)

		assert.Equal(
			t,
			map[string]string{"PG_VERSION": "17\n", "base/1/1259": "heap-bytes"},
			readTarEntries(t, backupArchive),
		)
	})
}
