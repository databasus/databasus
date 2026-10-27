package usecases_logical

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

const (
	dumpToolError    = "error: query failed: ERROR:  permission denied for sequence orders_id_seq"
	encryptionMarker = "encryption-marker-7f3a"
)

// stubDumpEngine opens a dump source of one engine whose dump tool is a stub running dumpScript.
type stubDumpEngine struct {
	name           string
	openDumpSource func(t *testing.T, dumpScript string) logicalDumpSource
}

func getStubDumpEngines() []stubDumpEngine {
	return []stubDumpEngine{
		{name: "PostgreSQL", openDumpSource: openStubPostgresqlDumpSource},
		{name: "MySQL", openDumpSource: openStubMysqlDumpSource},
		{name: "MariaDB", openDumpSource: openStubMariadbDumpSource},
		{name: "MongoDB", openDumpSource: openStubMongodbDumpSource},
	}
}

func TestStreamDumpToStorage(t *testing.T) {
	t.Parallel()

	for _, engine := range getStubDumpEngines() {
		t.Run(engine.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reports the dump tool error when the tool exits before producing data", func(t *testing.T) {
				t.Parallel()
				// The error reaches the error output only after the tool exited, which is how a
				// fast-exiting tool looks when the reader of its output is scheduled late.
				dumpSource := engine.openDumpSource(t, "(sleep 0.2; echo '"+dumpToolError+"' >&2) >/dev/null &\nexit 1")

				_, err := GetCreateBackupUsecase().streamDumpToStorage(
					t.Context(), newStubBackupStream(t, dumpSource, &discardingBackupFileStore{}, false),
				)

				require.Error(t, err)
				assert.Contains(t, err.Error(), dumpToolError)
				assert.NotContains(t, err.Error(), "save to storage")
			})

			t.Run("reports the storage error when the storage fails at once", func(t *testing.T) {
				t.Parallel()
				storageErr := errors.New("storage is unreachable")
				dumpSource := engine.openDumpSource(t, "head -c 4194304 /dev/zero")

				_, err := GetCreateBackupUsecase().streamDumpToStorage(t.Context(), newStubBackupStream(
					t, dumpSource, &failingBackupFileStore{failureErr: storageErr, isFailingBeforeReading: true}, false,
				))

				assertStorageErrorReported(t, err, storageErr)
			})

			t.Run(
				"reports a cancelled backup when it is cancelled while the storage stopped reading",
				func(t *testing.T) {
					t.Parallel()
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					fileStore := &stalledBackupFileStore{hasReadData: atomic.Bool{}}
					dumpSource := engine.openDumpSource(t, "head -c 4194304 /dev/zero")
					backupErrors := make(chan error, 1)

					go func() {
						_, err := GetCreateBackupUsecase().streamDumpToStorage(
							ctx, newStubBackupStream(t, dumpSource, fileStore, false),
						)
						backupErrors <- err
					}()

					testwait.Until(t, "the storage read the first byte of the dump", func(context.Context) error {
						if !fileStore.hasReadData.Load() {
							return errors.New("the storage has read nothing yet")
						}

						return nil
					})
					cancel()

					var backupErr error

					testwait.Until(t, "the cancelled backup returned", func(context.Context) error {
						select {
						case backupErr = <-backupErrors:
							return nil
						default:
							return errors.New("the backup is still running")
						}
					})
					require.EqualError(t, backupErr, "backup cancelled")
				},
			)
		})
	}

	t.Run("stores an encrypted backup without the dumped text", func(t *testing.T) {
		t.Parallel()
		stream, storedFile := newLocalStorageBackupStream(
			t,
			openStubPostgresqlDumpSource(t, "printf '"+encryptionMarker+"'"),
		)

		artifacts, err := GetCreateBackupUsecase().streamDumpToStorage(t.Context(), stream)

		require.NoError(t, err)
		assert.Equal(t, backups_core_enums.BackupEncryptionEncrypted, artifacts.Metadata.Encryption)
		storedBytes := readStoredFile(t, storedFile)
		assert.NotEmpty(t, storedBytes)
		assert.NotContains(t, string(storedBytes), encryptionMarker)
	})

	t.Run("stores two encrypted backups of one dump as different files with different salts", func(t *testing.T) {
		t.Parallel()
		firstStream, firstFile := newLocalStorageBackupStream(
			t,
			openStubPostgresqlDumpSource(t, "printf 'unchanged dump'"),
		)
		secondStream, secondFile := newLocalStorageBackupStream(
			t,
			openStubPostgresqlDumpSource(t, "printf 'unchanged dump'"),
		)

		firstArtifacts, err := GetCreateBackupUsecase().streamDumpToStorage(t.Context(), firstStream)
		require.NoError(t, err)
		secondArtifacts, err := GetCreateBackupUsecase().streamDumpToStorage(t.Context(), secondStream)
		require.NoError(t, err)

		assert.NotEqual(t, readStoredFile(t, firstFile), readStoredFile(t, secondFile))
		assert.NotEqual(t, *firstArtifacts.Metadata.EncryptionSalt, *secondArtifacts.Metadata.EncryptionSalt)
		assert.Equal(t, decodeBase64(t, *firstArtifacts.Metadata.EncryptionSalt), readStoredFile(t, firstFile)[8:40])
	})

	t.Run("reports the stored size of a MySQL dump after zstd when the dump completes", func(t *testing.T) {
		t.Parallel()
		var lastReportedMb float64
		stream := newStubBackupStream(
			t, openStubMysqlDumpSource(t, "head -c 3145728 /dev/zero"), &discardingBackupFileStore{}, false,
		)
		stream.progressListener = func(completedMBs float64) { lastReportedMb = completedMBs }

		_, err := GetCreateBackupUsecase().streamDumpToStorage(t.Context(), stream)

		require.NoError(t, err)
		assert.Positive(t, lastReportedMb)
		assert.Less(t, lastReportedMb, 1.0, "3 MB of zeros compress far below 1 MB")
	})
}

func newStubBackupStream(
	t *testing.T,
	dumpSource logicalDumpSource,
	fileStore backups_core_logical.BackupFileStore,
	isEncrypted bool,
) backupStream {
	t.Helper()

	encryption := backups.EncryptionNone
	if isEncrypted {
		encryption = backups.EncryptionEncrypted
	}

	return backupStream{
		backup: newTestBackup(),
		backupConfig: backups.LogicalBackupConfig{
			BackupConfig: backups.BackupConfig{
				Encryption: encryption,
			},
		},
		dumpSource:       dumpSource,
		fileStore:        fileStore,
		progressListener: func(float64) {},
		logger:           slog.New(slog.DiscardHandler),
	}
}

// newLocalStorageBackupStream returns an encrypted backup into a local storage of the storages
// module, and the reference of the file it stores.
func newLocalStorageBackupStream(
	t *testing.T,
	dumpSource logicalDumpSource,
) (backupStream, storages.StoredFileReference) {
	t.Helper()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	storage := storages_bridge.CreateTestStorage(workspace.ID)

	stream := newStubBackupStream(t, dumpSource, storages_bridge.GetModule(), true)
	stream.backup.StorageID = storage.ID
	storedFile := storages.StoredFileReference{StorageID: storage.ID, FileName: stream.backup.FileName}

	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		_ = storages_bridge.DeleteTestFile(ctx, storedFile)
		storages_bridge.RemoveTestStorage(ctx, storage.ID)
		workspaces_testing.RemoveTestWorkspace(ctx, workspace)
	})

	return stream, storedFile
}

func readStoredFile(t *testing.T, storedFile storages.StoredFileReference) []byte {
	t.Helper()

	reader, err := storages_bridge.OpenTestFile(t.Context(), storedFile)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	storedBytes, err := io.ReadAll(reader)
	require.NoError(t, err)

	return storedBytes
}

func decodeBase64(t *testing.T, value string) []byte {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(value)
	require.NoError(t, err)

	return decoded
}

func openStubPostgresqlDumpSource(t *testing.T, dumpScript string) logicalDumpSource {
	t.Helper()

	clientTools := pgshared.ClientTools{Root: t.TempDir()}
	toolstub.Write(t, clientTools.GetExecutable(pgshared.Version16, pgshared.ExecutablePgDump), toolstub.Spec{
		IsReadingStdin: false,
		Script:         dumpScript,
	})

	tunnel, err := pglogical.NewSource(pglogical.Settings{
		Host: "127.0.0.1", Port: 5432, Username: "backup_user", Password: "password", Database: "orders",
	}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &postgresqlDumpSource{
		tunnel:     tunnel,
		dumpFilter: pglogical.DumpFilter{},
		spec: pglogical.DumpSpec{
			ClientTools:      clientTools,
			Version:          pgshared.Version16,
			IncludeSchemas:   nil,
			ExcludeTables:    nil,
			ExcludeTableData: nil,
			Logger:           slog.New(slog.DiscardHandler),
		},
		logger: slog.New(slog.DiscardHandler),
	}
}

func openStubMysqlDumpSource(t *testing.T, dumpScript string) logicalDumpSource {
	t.Helper()

	clientToolsRoot := t.TempDir()
	binDir := filepath.Join(clientToolsRoot, "mysql", "mysql-"+string(mysql_engine.Version80), "bin")
	writeStubDumpAndClientTools(
		t,
		binDir,
		string(mysql_engine.ExecutableMysqldump),
		string(mysql_engine.ExecutableMysql),
		dumpScript,
	)

	tunnel, err := mysql_engine.NewSource(mysql_engine.Settings{
		Host: "127.0.0.1", Port: 3306, Username: "backup_user", Password: "password", Database: "orders",
	}, mysql_engine.ClientTools{Root: clientToolsRoot}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mysqlDumpSource{
		tunnel: tunnel,
		spec: mysql_engine.DumpSpec{
			Version:       mysql_engine.Version80,
			ExcludeTables: nil,
			Privileges:    nil,
			Logger:        slog.New(slog.DiscardHandler),
		},
		logger: slog.New(slog.DiscardHandler),
	}
}

func openStubMariadbDumpSource(t *testing.T, dumpScript string) logicalDumpSource {
	t.Helper()

	clientToolsRoot := t.TempDir()
	clientVersion := mariadb_engine.GetClientVersionForServer(mariadb_engine.Version1011)
	binDir := filepath.Join(clientToolsRoot, "mariadb", "mariadb-"+string(clientVersion), "bin")
	writeStubDumpAndClientTools(
		t,
		binDir,
		string(mariadb_engine.ExecutableMariadbDump),
		string(mariadb_engine.ExecutableMariadb),
		dumpScript,
	)

	tunnel, err := mariadb_engine.NewSource(mariadb_engine.Settings{
		Host: "127.0.0.1", Port: 3306, Username: "backup_user", Password: "password", Database: "orders",
	}, mariadb_engine.ClientTools{Root: clientToolsRoot}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mariadbDumpSource{
		tunnel: tunnel,
		spec: mariadb_engine.DumpSpec{
			Version:         mariadb_engine.Version1011,
			ExcludeTables:   nil,
			Privileges:      nil,
			IsExcludeEvents: false,
			Logger:          slog.New(slog.DiscardHandler),
		},
		logger: slog.New(slog.DiscardHandler),
	}
}

func openStubMongodbDumpSource(t *testing.T, dumpScript string) logicalDumpSource {
	t.Helper()

	clientTools := mongodb_engine.ClientTools{Root: t.TempDir()}
	toolstub.Write(t, clientTools.GetExecutable(mongodb_engine.ExecutableMongodump), toolstub.Spec{
		IsReadingStdin: false,
		Script:         dumpScript,
	})

	tunnel, err := mongodb_engine.NewSource(mongodb_engine.Settings{
		Host: "127.0.0.1", Port: 27017, Username: "backup_user", Password: "password", Database: "orders",
	}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mongodbDumpSource{
		tunnel: tunnel,
		spec: mongodb_engine.DumpSpec{
			ClientTools:         clientTools,
			ExcludeCollections:  nil,
			ParallelCollections: 1,
			Logger:              slog.New(slog.DiscardHandler),
		},
		logger: slog.New(slog.DiscardHandler),
	}
}

// writeStubDumpAndClientTools writes the dump tool and the client of a MySQL family bundle, which
// counts only when it holds both. The network compression probe of the dump passes, because it
// names a table that cannot exist and the stub exits successfully on it.
func writeStubDumpAndClientTools(t *testing.T, binDir, dumpTool, clientTool, dumpScript string) {
	t.Helper()

	toolstub.Write(t, filepath.Join(binDir, dumpTool), toolstub.Spec{
		IsReadingStdin: false,
		Script:         `case "$*" in *--no-data*) exit 0;; esac` + "\n" + dumpScript,
	})
	toolstub.Write(t, filepath.Join(binDir, clientTool), toolstub.Spec{IsReadingStdin: true, Script: ""})
}

// discardingBackupFileStore reads every write to the end and keeps nothing, for tests whose backup
// fails for a reason other than the storage.
type discardingBackupFileStore struct{}

func (s *discardingBackupFileStore) WriteFile(
	_ context.Context,
	_ storages.StoredFileReference,
	file io.Reader,
) (storages.WriteReceipt, error) {
	_, err := io.Copy(io.Discard, file)

	return storages.WriteReceipt{}, err
}

func (s *discardingBackupFileStore) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	return storages_bridge.StartFakeBackgroundWrite(ctx, s.WriteFile, spec)
}

// stalledBackupFileStore reads the first byte of a write and then stops reading without closing the
// stream, like a storage stuck on the network. It returns only when the backup context ends.
type stalledBackupFileStore struct {
	hasReadData atomic.Bool
}

func (s *stalledBackupFileStore) WriteFile(
	_ context.Context,
	_ storages.StoredFileReference,
	_ io.Reader,
) (storages.WriteReceipt, error) {
	return storages.WriteReceipt{}, errors.New("the stalled store only writes in the background")
}

func (s *stalledBackupFileStore) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	write := storages.BackgroundWrite{
		Errors:   make(chan error, 1),
		Receipts: make(chan storages.WriteReceipt, 1),
	}

	go func() {
		_, err := io.ReadFull(spec.Source, make([]byte, 1))
		if err == nil {
			s.hasReadData.Store(true)
			<-ctx.Done()
			err = context.Cause(ctx)
		}

		write.Receipts <- storages.WriteReceipt{}
		write.Errors <- err
	}()

	return write
}
