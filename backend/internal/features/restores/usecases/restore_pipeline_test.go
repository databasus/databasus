package usecases

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
)

const (
	backupStreamFailure = "backup stream from storage failed mid-restore"
	restoreToolError    = "ERROR 1064 (42000) at line 53465: You have an error in your SQL syntax"
	encryptionMarker    = "encryption-marker-7f3a"
)

// stubRestoreEngine builds a restore target of one engine whose restore tool is a stub.
type stubRestoreEngine struct {
	name             string
	newRestoreTarget func(t *testing.T, spec toolstub.Spec) (logicalRestoreTarget, *toolstub.Stub)
}

func getStubRestoreEngines() []stubRestoreEngine {
	return []stubRestoreEngine{
		{name: "PostgreSQL", newRestoreTarget: newStubPostgresqlRestoreTarget},
		{name: "MySQL", newRestoreTarget: newStubMysqlRestoreTarget},
		{name: "MariaDB", newRestoreTarget: newStubMariadbRestoreTarget},
		{name: "MongoDB", newRestoreTarget: newStubMongodbRestoreTarget},
	}
}

func TestRestoreFromStorage(t *testing.T) {
	t.Parallel()

	for _, engine := range getStubRestoreEngines() {
		t.Run(engine.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reports the tool error written after the tool exited when the stream is whole", func(t *testing.T) {
				t.Parallel()
				restoreTarget, _ := engine.newRestoreTarget(t, toolstub.Spec{
					IsReadingStdin: true,
					Script:         "(sleep 0.2; echo '" + restoreToolError + "' >&2) &\nexit 1",
				})
				backup := newUnencryptedTestBackup()
				storedFile := encodeStoredFile(t, restoreTarget, []byte("INSERT INTO orders VALUES (1);\n"))

				err := GetRestoreBackupUsecase().restoreFromStorage(
					t.Context(), newStubRestoreRun(backup, &memoryStorageReader{file: storedFile}), restoreTarget,
				)

				require.Error(t, err)
				assert.NotContains(t, err.Error(), backupStreamFailure)
				assert.Contains(t, err.Error(), "SQL syntax")
			})

			t.Run(
				"reports the backup stream failure instead of the tool error when the storage read fails",
				func(t *testing.T) {
					t.Parallel()
					restoreTarget, _ := engine.newRestoreTarget(t, toolstub.Spec{
						IsReadingStdin: true,
						Script:         "echo '" + restoreToolError + "' >&2; exit 1",
					})
					backup := newUnencryptedTestBackup()
					storedFile := encodeStoredFile(
						t,
						restoreTarget,
						bytes.Repeat([]byte("INSERT INTO orders VALUES (1);\n"), 100_000),
					)

					err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(backup, &memoryStorageReader{
						file:         storedFile[:len(storedFile)/2],
						failAtEndErr: errors.New("connection reset by peer"),
					}), restoreTarget)

					require.Error(t, err)
					assert.Contains(t, err.Error(), backupStreamFailure)
					assert.NotContains(t, err.Error(), "SQL syntax")
				},
			)

			t.Run("reports a cancelled restore when it is cancelled while the tool reads", func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				restoreTarget, _ := engine.newRestoreTarget(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
				storageReader := &stallingStorageReader{
					file:          encodeStoredFile(t, restoreTarget, []byte("INSERT INTO orders VALUES (1);\n")),
					hasServedFile: atomic.Bool{},
				}
				restoreErrors := make(chan error, 1)

				go func() {
					restoreErrors <- GetRestoreBackupUsecase().restoreFromStorage(
						ctx, newStubRestoreRun(newUnencryptedTestBackup(), storageReader), restoreTarget,
					)
				}()

				testwait.Until(t, "the restore read the whole stored file", func(context.Context) error {
					if !storageReader.hasServedFile.Load() {
						return errors.New("the stored file is still being read")
					}

					return nil
				})
				cancel()

				var restoreErr error

				testwait.Until(t, "the cancelled restore returned", func(context.Context) error {
					select {
					case restoreErr = <-restoreErrors:
						return nil
					default:
						return errors.New("the restore is still running")
					}
				})
				require.EqualError(t, restoreErr, "restore cancelled")
			})
		})
	}

	t.Run(
		"reports the backup stream failure instead of the syntax error when the zstd stream is truncated",
		func(t *testing.T) {
			t.Parallel()
			restoreTarget, _ := newStubMysqlRestoreTarget(t, toolstub.Spec{
				IsReadingStdin: true,
				Script:         "echo '" + restoreToolError + "' >&2; exit 1",
			})
			storedFile := encodeStoredFile(t, restoreTarget, bytes.Repeat(
				[]byte("INSERT INTO orders VALUES ('2020-07-13 18:49:41');\n"), 200_000,
			))

			err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(
				newUnencryptedTestBackup(), &memoryStorageReader{file: storedFile[:len(storedFile)*3/4]},
			), restoreTarget)

			require.Error(t, err)
			assert.Contains(t, err.Error(), backupStreamFailure)
			assert.NotContains(t, err.Error(), "SQL syntax")
		},
	)

	t.Run("feeds the restore tool the whole dump of an encrypted backup", func(t *testing.T) {
		t.Parallel()
		restoreTarget, pgRestore := newStubPostgresqlRestoreTarget(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		dump := []byte("PGDMP " + encryptionMarker)
		backup, encryptedFile := encryptTestBackup(t, dump)
		storageReader := writeToLocalStorage(t, backup, encryptedFile)

		err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(backup, storageReader), restoreTarget)

		require.NoError(t, err)
		assert.NotContains(t, string(encryptedFile), encryptionMarker)
		assert.Equal(t, dump, pgRestore.GetStdin(t))
	})

	t.Run("reports the backup stream failure when a byte of the encrypted file is changed", func(t *testing.T) {
		t.Parallel()
		restoreTarget, _ := newStubPostgresqlRestoreTarget(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		backup, encryptedFile := encryptTestBackup(t, bytes.Repeat([]byte("PGDMP"), 1000))
		encryptedFile[len(encryptedFile)/2] ^= 0xFF
		storageReader := writeToLocalStorage(t, backup, encryptedFile)

		err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(backup, storageReader), restoreTarget)

		require.Error(t, err)
		assert.Contains(t, err.Error(), backupStreamFailure)
	})

	t.Run("reports the backup stream failure when the encrypted file is cut inside a chunk", func(t *testing.T) {
		t.Parallel()
		restoreTarget, _ := newStubPostgresqlRestoreTarget(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		backup, encryptedFile := encryptTestBackup(t, bytes.Repeat([]byte("PGDMP"), 1000))
		storageReader := writeToLocalStorage(t, backup, encryptedFile[:len(encryptedFile)-2500])

		err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(backup, storageReader), restoreTarget)

		require.Error(t, err)
		assert.Contains(t, err.Error(), backupStreamFailure)
	})

	t.Run("fails before the restore tool reads a byte when the file is another backup's", func(t *testing.T) {
		t.Parallel()
		restoreTarget, pgRestore := newStubPostgresqlRestoreTarget(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		backup, _ := encryptTestBackup(t, []byte("PGDMP first backup"))
		_, otherBackupFile := encryptTestBackup(t, []byte("PGDMP second backup"))
		storageReader := writeToLocalStorage(t, backup, otherBackupFile)

		err := GetRestoreBackupUsecase().restoreFromStorage(t.Context(), newStubRestoreRun(backup, storageReader), restoreTarget)

		require.ErrorIs(t, err, crypto.ErrStreamHeaderMismatch)
		assert.Empty(t, pgRestore.GetInvocations(t))
		assert.Empty(t, pgRestore.GetStdin(t))
	})
}

func newStubRestoreRun(backup *backups_core_logical.LogicalBackup, storageReader storages.StorageReader) restoreRun {
	return restoreRun{
		source:        restores_core.SourceDatabase{},
		target:        restores_core.RestoreBackupRequest{},
		backup:        backup,
		storageReader: storageReader,
		options:       restores_core.RestoreOptions{},
		logger:        slog.New(slog.DiscardHandler),
	}
}

func newUnencryptedTestBackup() *backups_core_logical.LogicalBackup {
	backupID := uuid.New()

	return &backups_core_logical.LogicalBackup{
		ID:         backupID,
		FileName:   "test-backup-" + backupID.String(),
		Encryption: backups_core_enums.BackupEncryptionNone,
	}
}

// encryptTestBackup encrypts dump the way an encrypted backup stores it and returns the backup row
// that names the stored file and its header.
func encryptTestBackup(t *testing.T, dump []byte) (*backups_core_logical.LogicalBackup, []byte) {
	t.Helper()

	backup := newUnencryptedTestBackup()

	var encryptedFile bytes.Buffer

	encryptingWriter, header, err := crypto.NewEncryptingWriter(&encryptedFile, encryption.GetMasterKey(), backup.ID)
	require.NoError(t, err)
	_, err = encryptingWriter.Write(dump)
	require.NoError(t, err)
	require.NoError(t, encryptingWriter.Close())

	saltBase64 := base64.StdEncoding.EncodeToString(header.Salt)
	nonceBase64 := base64.StdEncoding.EncodeToString(header.Nonce)
	backup.Encryption = backups_core_enums.BackupEncryptionEncrypted
	backup.EncryptionSalt = &saltBase64
	backup.EncryptionIV = &nonceBase64

	return backup, encryptedFile.Bytes()
}

// writeToLocalStorage stores the file of backup in a local storage of the storages module and
// returns that storage's reader.
func writeToLocalStorage(
	t *testing.T,
	backup *backups_core_logical.LogicalBackup,
	storedFile []byte,
) storages.StorageReader {
	t.Helper()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	storage := storages_bridge.CreateTestStorage(workspace.ID)
	reference := storages.StoredFileReference{StorageID: storage.ID, FileName: backup.FileName}

	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		_ = storages_bridge.DeleteTestFile(ctx, reference)
		storages_bridge.RemoveTestStorage(ctx, storage.ID)
		workspaces_testing.RemoveTestWorkspace(ctx, workspace)
	})

	require.NoError(t, storages_bridge.WriteTestFile(t.Context(), reference, bytes.NewReader(storedFile)))

	storageReader, err := storages_bridge.GetModule().GetStorageReader(t.Context(), storage.ID)
	require.NoError(t, err)

	return storageReader
}

// encodeStoredFile returns dump in the stored format of the target's engine.
func encodeStoredFile(t *testing.T, restoreTarget logicalRestoreTarget, dump []byte) []byte {
	t.Helper()

	if !restoreTarget.GetStoredCompression().isZstd {
		return dump
	}

	var compressedDump bytes.Buffer

	zstdWriter, err := zstd.NewWriter(&compressedDump)
	require.NoError(t, err)
	_, err = zstdWriter.Write(dump)
	require.NoError(t, err)
	require.NoError(t, zstdWriter.Close())

	return compressedDump.Bytes()
}

// memoryStorageReader serves one file from memory and fails the read at its end with failAtEndErr,
// like a download severed mid-file. Without failAtEndErr the file ends normally.
type memoryStorageReader struct {
	file         []byte
	failAtEndErr error
}

func (r *memoryStorageReader) OpenFile(context.Context, string) (io.ReadCloser, error) {
	var fileReader io.Reader = bytes.NewReader(r.file)
	if r.failAtEndErr != nil {
		fileReader = io.MultiReader(fileReader, failingReader{err: r.failAtEndErr})
	}

	return io.NopCloser(fileReader), nil
}

// stallingStorageReader serves one file from memory and then, instead of ending it, blocks the next
// read until the restore context ends, like a download that stopped arriving.
type stallingStorageReader struct {
	file          []byte
	hasServedFile atomic.Bool
}

func (r *stallingStorageReader) OpenFile(ctx context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(io.MultiReader(bytes.NewReader(r.file), stallingReader{ctx: ctx, reader: r})), nil
}

type stallingReader struct {
	ctx    context.Context
	reader *stallingStorageReader
}

func (r stallingReader) Read([]byte) (int, error) {
	r.reader.hasServedFile.Store(true)
	<-r.ctx.Done()

	return 0, context.Cause(r.ctx)
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func newStubPostgresqlRestoreTarget(t *testing.T, spec toolstub.Spec) (logicalRestoreTarget, *toolstub.Stub) {
	t.Helper()

	clientTools := pgshared.ClientTools{Root: t.TempDir()}
	pgRestore := toolstub.Write(t, clientTools.GetExecutable(pgshared.Version16, pgshared.ExecutablePgRestore), spec)

	return &postgresqlRestoreTarget{
		source: pglogical.NewSource(pglogical.Settings{
			Host: "127.0.0.1", Port: 5432, Username: "restore_user", Password: "password", Database: "orders",
		}),
		spec: pglogical.RestoreSpec{
			ClientTools:         clientTools,
			Version:             pgshared.Version16,
			SourceDatabaseName:  "orders",
			ParallelJobs:        1,
			IsTimescaleDB:       false,
			IsRestoreOwnership:  false,
			IsRestorePrivileges: false,
			IsExcludeExtensions: false,
			IsSkipUserMappings:  false,
			TempDir:             t.TempDir(),
			Logger:              slog.New(slog.DiscardHandler),
		},
	}, pgRestore
}

func newStubMysqlRestoreTarget(t *testing.T, spec toolstub.Spec) (logicalRestoreTarget, *toolstub.Stub) {
	t.Helper()

	clientToolsRoot := t.TempDir()
	binDir := filepath.Join(clientToolsRoot, "mysql", "mysql-"+string(mysql_engine.Version80), "bin")
	toolstub.Write(t, filepath.Join(binDir, string(mysql_engine.ExecutableMysqldump)), toolstub.Spec{})
	mysqlClient := toolstub.Write(t, filepath.Join(binDir, string(mysql_engine.ExecutableMysql)), spec)

	return &mysqlRestoreTarget{
		source: mysql_engine.NewSource(mysql_engine.Settings{
			Host: "127.0.0.1", Port: 3306, Username: "restore_user", Password: "password", Database: "orders",
		}, mysql_engine.ClientTools{Root: clientToolsRoot}),
		spec: mysql_engine.RestoreSpec{
			Version:            mysql_engine.Version80,
			SourceDatabaseName: "orders",
			Logger:             slog.New(slog.DiscardHandler),
		},
	}, mysqlClient
}

func newStubMariadbRestoreTarget(t *testing.T, spec toolstub.Spec) (logicalRestoreTarget, *toolstub.Stub) {
	t.Helper()

	clientToolsRoot := t.TempDir()
	clientVersion := mariadb_engine.GetClientVersionForServer(mariadb_engine.Version1011)
	binDir := filepath.Join(clientToolsRoot, "mariadb", "mariadb-"+string(clientVersion), "bin")
	toolstub.Write(t, filepath.Join(binDir, string(mariadb_engine.ExecutableMariadbDump)), toolstub.Spec{})
	mariadbClient := toolstub.Write(t, filepath.Join(binDir, string(mariadb_engine.ExecutableMariadb)), spec)

	return &mariadbRestoreTarget{
		source: mariadb_engine.NewSource(mariadb_engine.Settings{
			Host: "127.0.0.1", Port: 3306, Username: "restore_user", Password: "password", Database: "orders",
		}, mariadb_engine.ClientTools{Root: clientToolsRoot}),
		spec: mariadb_engine.RestoreSpec{
			Version:             mariadb_engine.Version1011,
			SourceDatabaseName:  "orders",
			IsSkipGaleraDisable: false,
			Logger:              slog.New(slog.DiscardHandler),
		},
	}, mariadbClient
}

func newStubMongodbRestoreTarget(t *testing.T, spec toolstub.Spec) (logicalRestoreTarget, *toolstub.Stub) {
	t.Helper()

	clientTools := mongodb_engine.ClientTools{Root: t.TempDir()}
	mongorestore := toolstub.Write(t, clientTools.GetExecutable(mongodb_engine.ExecutableMongorestore), spec)

	return &mongodbRestoreTarget{
		source: mongodb_engine.NewSource(mongodb_engine.Settings{
			Host: "127.0.0.1", Port: 27017, Username: "restore_user", Password: "password", Database: "orders",
		}),
		spec: mongodb_engine.RestoreSpec{
			ClientTools:        clientTools,
			SourceDatabaseName: "orders",
			InsertionWorkers:   1,
			Logger:             slog.New(slog.DiscardHandler),
		},
	}, mongorestore
}
