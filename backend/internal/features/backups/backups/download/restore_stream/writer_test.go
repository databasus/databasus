package restore_stream_test

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/features/backups/backups/download/restore_stream"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func Test_Writer_Write_ProducesCombineableLayoutWithIntegrityManifest(t *testing.T) {
	storage := createTestStorage(t)
	encryptor := encryption.GetFieldEncryptor()
	ctx := t.Context()

	pgVersionBody := "17\n"
	fullArtifact := buildTarZst(t, map[string]string{
		"PG_VERSION":  pgVersionBody,
		"base/1/1259": "heap-bytes",
	})
	manifestBody := []byte(`{ "PostgreSQL-Backup-Manifest-Version": 2 }`)
	walBody := "WALSEGMENT-0003-"
	walCompressed := zstdBytes(t, []byte(walBody))
	historyBody := "1\t0/3000060\tno recovery target specified\n"

	saveFile(t, ctx, storage, encryptor, "test-full", fullArtifact)
	saveFile(t, ctx, storage, encryptor, "test-full.manifest", manifestBody)
	saveFile(t, ctx, storage, encryptor, "test-wal-0003.zst", walCompressed)
	saveFile(t, ctx, storage, encryptor, "test-hist.history.zst", zstdBytes(t, []byte(historyBody)))
	saveFile(t, ctx, storage, encryptor, "test-hist.history.zst.metadata",
		mustJSON(t, physical_dto.PhysicalWalHistoryMetadata{Encryption: backups_core_enums.BackupEncryptionNone}))

	fullSizeMb := float64(len(fullArtifact)) / (1024 * 1024)
	set := &chain_view.RestoreSet{
		RootFull: &physical_models.PhysicalFullBackup{
			ID:               uuid.New(),
			StorageID:        storage.ID,
			FileName:         strPtr("test-full"),
			ManifestFileName: strPtr("test-full.manifest"),
			Encryption:       backups_core_enums.BackupEncryptionNone,
			Compression:      physical_enums.PhysicalBackupCompressionZstd,
			BackupSizeMb:     &fullSizeMb,
		},
		WalSegments: []*physical_models.PhysicalWalSegment{
			{
				ID:          uuid.New(),
				StorageID:   storage.ID,
				FileName:    strPtr("test-wal-0003.zst"),
				WalFilename: "000000010000000000000003",
				Encryption:  backups_core_enums.BackupEncryptionNone,
			},
		},
		HistoryFiles: []*physical_models.PhysicalWalHistoryFile{
			{
				ID:              uuid.New(),
				StorageID:       storage.ID,
				FileName:        "test-hist.history.zst",
				HistoryFilename: "00000002.history",
			},
		},
	}

	var out bytes.Buffer
	writer := restore_stream.NewWriter(storages.GetStorageService(), encryptor, logger.GetLogger())
	err := writer.Write(t.Context(), &out, set, "")
	require.NoError(t, err)

	entries := readTar(t, &out)

	assert.Equal(t, string(fullArtifact), entries["full/base.tar.zst"],
		"the full backup must ship as the stored compressed blob, never inflated")
	assert.Equal(t, string(manifestBody), entries["full/backup_manifest"])

	assert.Equal(t, string(walCompressed), entries["wal/000000010000000000000003.zst"],
		"wal must ship as stored (compressed) under <wal>.zst, never re-inflated")
	assert.Equal(t, historyBody, entries["wal/00000002.history"])

	manifestLines := entries["MANIFEST.sha256"]
	require.Contains(t, manifestLines, sha256Hex(string(fullArtifact))+"  full/base.tar.zst",
		"the integrity manifest hashes the compressed blob bytes")
	require.Contains(t, manifestLines, "  wal/000000010000000000000003.zst",
		"the integrity manifest hashes the compressed wal bytes")
	require.NotContains(t, manifestLines, "MANIFEST.sha256", "the integrity manifest must not list itself")
}

func createTestStorage(t *testing.T) *storages.Storage {
	t.Helper()

	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(
		t.Context(),
		"Restore Stream Test "+uuid.NewString(),
		user,
		router,
	)

	storage := storages.CreateTestStorage(workspace.ID)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), storage.ID) })

	return storage
}

func saveFile(
	t *testing.T,
	ctx context.Context,
	storage *storages.Storage,
	encryptor encryption.FieldEncryptor,
	name string,
	body []byte,
) {
	t.Helper()

	err := storage.SaveFile(ctx, encryptor, logger.GetLogger(), name, bytes.NewReader(body))
	require.NoError(t, err)

	t.Cleanup(func() { _ = storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), name) })
}

func buildTarZst(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var raw bytes.Buffer
	tarWriter := tar.NewWriter(&raw)

	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o600,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())

	return zstdBytes(t, raw.Bytes())
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

func readTar(t *testing.T, r io.Reader) map[string]string {
	t.Helper()

	entries := make(map[string]string)
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Typeflag != tar.TypeReg {
			continue
		}

		body, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		entries[header.Name] = string(body)
	}

	return entries
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	body, err := json.Marshal(v)
	require.NoError(t, err)

	return body
}

func sha256Hex(body string) string {
	sum := sha256.Sum256([]byte(body))

	return hex.EncodeToString(sum[:])
}

func strPtr(s string) *string {
	return &s
}
