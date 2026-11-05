package restoring_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestPhysicalRestoreIntegrity(t *testing.T) {
	t.Parallel()

	t.Run(
		"streams compressed backup and WAL bytes with plaintext history and an integrity manifest",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			timeline := newRestoreTimeline()
			full := catalog.seedRestoreFull(
				t,
				restoreFullSpec{StartLSN: lsnAt(2), StopLSN: lsnAt(3), CompletedAt: timeline.at(0)},
			)

			fullArtifact := zstdTar(t, map[string]string{"PG_VERSION": "17\n", "base/1/1259": "heap-bytes"})
			manifestBody := []byte(`{ "PostgreSQL-Backup-Manifest-Version": 2 }`)
			catalog.publish(t, *full.FileName, fullArtifact)
			catalog.publish(t, *full.ManifestFileName, manifestBody)
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical SET backup_size_mb=? WHERE id=?",
				float64(len(fullArtifact))/(1024*1024),
				full.ID,
			)

			wal := catalog.seedRestoreWal(t, restoreWalSpec{Segment: 3, ReceivedAt: timeline.at(5)})
			walCompressed := zstdBytes(t, []byte("WALSEGMENT-0003-"))
			catalog.publish(t, *wal.FileName, walCompressed)

			historyBody := "1\t0/3000060\tno recovery target specified\n"
			history := catalog.App.InsertPhysicalWalHistoryFile(
				t,
				apptest.PhysicalWalHistoryFileSeed{
					DatabaseID:      catalog.DatabaseID,
					StorageID:       catalog.StorageID,
					TimelineID:      2,
					FileName:        "test-hist.history.zst",
					HistoryFilename: "00000002.history",
				},
			)
			catalog.publish(t, history.FileName, zstdBytes(t, []byte(historyBody)))
			catalog.publish(t, history.FileName+".metadata", []byte(`{"encryption":"NONE"}`))

			entries := catalog.downloadRestore(t, restoreRequest{})

			assert.Equal(t, string(fullArtifact), entries["full/base.tar.zst"])
			assert.Equal(t, string(manifestBody), entries["full/backup_manifest"])
			assert.Equal(t, string(walCompressed), entries["wal/000000010000000000000003.zst"])
			assert.Equal(t, historyBody, entries["wal/00000002.history"])
			manifestLines := entries["MANIFEST.sha256"]
			require.Contains(t, manifestLines, sha256Hex(string(fullArtifact))+"  full/base.tar.zst")
			require.Contains(t, manifestLines, sha256Hex(string(walCompressed))+"  wal/000000010000000000000003.zst")
			require.NotContains(t, manifestLines, "MANIFEST.sha256")
		},
	)
}

func zstdBytes(t *testing.T, body []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(body)
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	return compressed.Bytes()
}

func sha256Hex(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
