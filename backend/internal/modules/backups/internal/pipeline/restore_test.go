package pipeline

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"path/filepath"
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
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/storages"
)

const (
	backupStreamFailure = "backup stream from storage failed mid-restore"
	restoreToolError    = "ERROR 1064 (42000) at line 53465: You have an error in your SQL syntax"
)

// stubRestoreEngine opens a restore sink of one engine whose restore tool is a stub.
type stubRestoreEngine struct {
	name        string
	compression storedCompression
	openSink    func(t *testing.T, spec toolstub.Spec) (logicalRestoreSink, *toolstub.Stub)
}

func getStubRestoreEngines() []stubRestoreEngine {
	return []stubRestoreEngine{
		{name: "PostgreSQL", compression: getStoredCompressionNone(), openSink: openStubPostgresqlRestoreSink},
		{name: "MySQL", compression: (&mysqlEngine{}).GetStoredCompression(), openSink: openStubMysqlRestoreSink},
		{
			name:        "MariaDB",
			compression: (&mariadbEngine{}).GetStoredCompression(),
			openSink:    openStubMariadbRestoreSink,
		},
		{name: "MongoDB", compression: getStoredCompressionNone(), openSink: openStubMongodbRestoreSink},
	}
}

func TestRestoreFromStorage(t *testing.T) {
	t.Parallel()

	for _, engine := range getStubRestoreEngines() {
		t.Run(engine.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reports the tool error written after the tool exited when the stream is whole", func(t *testing.T) {
				t.Parallel()
				sink, _ := engine.openSink(t, toolstub.Spec{
					IsReadingStdin: true,
					Script:         "(sleep 0.2; echo '" + restoreToolError + "' >&2) &\nexit 1",
				})
				storedFile := encodeStoredFile(t, engine.compression, []byte("INSERT INTO orders VALUES (1);\n"))
				restorePipeline := newTestPipeline(t, newDiscardingStorage())

				err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
					newUnencryptedStoredBackup(),
					&memoryStorageReader{file: storedFile, failAtEndErr: nil},
					sink,
					engine.compression,
				))

				require.Error(t, err)
				assert.NotContains(t, err.Error(), backupStreamFailure)
				assert.Contains(t, err.Error(), "SQL syntax")
			})

			t.Run(
				"reports the backup stream failure instead of the tool error when the storage read fails",
				func(t *testing.T) {
					t.Parallel()
					sink, _ := engine.openSink(t, toolstub.Spec{
						IsReadingStdin: true,
						Script:         "echo '" + restoreToolError + "' >&2; exit 1",
					})
					storedFile := encodeStoredFile(
						t,
						engine.compression,
						bytes.Repeat([]byte("INSERT INTO orders VALUES (1);\n"), 100_000),
					)
					restorePipeline := newTestPipeline(t, newDiscardingStorage())

					err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
						newUnencryptedStoredBackup(),
						&memoryStorageReader{
							file:         storedFile[:len(storedFile)/2],
							failAtEndErr: errors.New("connection reset by peer"),
						},
						sink,
						engine.compression,
					))

					require.Error(t, err)
					assert.Contains(t, err.Error(), backupStreamFailure)
					assert.NotContains(t, err.Error(), "SQL syntax")
				},
			)

			t.Run("reports a cancelled restore when it is cancelled while the tool reads", func(t *testing.T) {
				t.Parallel()
				runCtx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)
				sink, _ := engine.openSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
				storageReader := &stallingStorageReader{
					file: encodeStoredFile(t, engine.compression, []byte("INSERT INTO orders VALUES (1);\n")),
				}
				restorePipeline := newTestPipeline(t, newDiscardingStorage())
				restoreErrors := make(chan error, 1)

				go func() {
					restoreErrors <- restorePipeline.restoreFromStorage(runCtx, newStubRestoreRun(
						newUnencryptedStoredBackup(), storageReader, sink, engine.compression,
					))
				}()

				testwait.Until(t, "the restore read the whole stored file", func(context.Context) error {
					if !storageReader.hasServedFile.Load() {
						return errors.New("the stored file is still being read")
					}

					return nil
				})
				cancel()

				require.ErrorIs(t, <-restoreErrors, runcontext.ErrRestoreCancelled)
			})
		})
	}

	t.Run(
		"reports the backup stream failure instead of the syntax error when the zstd stream is truncated",
		func(t *testing.T) {
			t.Parallel()
			compression := (&mysqlEngine{}).GetStoredCompression()
			sink, _ := openStubMysqlRestoreSink(t, toolstub.Spec{
				IsReadingStdin: true,
				Script:         "echo '" + restoreToolError + "' >&2; exit 1",
			})
			storedFile := encodeStoredFile(t, compression, bytes.Repeat(
				[]byte("INSERT INTO orders VALUES ('2020-07-13 18:49:41');\n"), 200_000,
			))

			err := newTestPipeline(t, newDiscardingStorage()).restoreFromStorage(t.Context(), newStubRestoreRun(
				newUnencryptedStoredBackup(),
				&memoryStorageReader{file: storedFile[:len(storedFile)*3/4], failAtEndErr: nil},
				sink,
				compression,
			))

			require.Error(t, err)
			assert.Contains(t, err.Error(), backupStreamFailure)
			assert.NotContains(t, err.Error(), "SQL syntax")
		},
	)

	t.Run("reports the storage failure when the archive download fails", func(t *testing.T) {
		t.Parallel()

		sink, pgRestore := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: false, Script: ""})
		postgresqlSink, isPostgresqlSink := sink.(*postgresqlRestoreSink)
		require.True(t, isPostgresqlSink)
		postgresqlSink.spec.ParallelJobs = 2

		restorePipeline := newTestPipeline(t, newDiscardingStorage())

		err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
			newUnencryptedStoredBackup(),
			&memoryStorageReader{
				file:         []byte("part of the archive"),
				failAtEndErr: errors.New("connection reset by peer"),
			},
			sink,
			getStoredCompressionNone(),
		))

		require.ErrorContains(t, err, "connection reset by peer")
		require.ErrorContains(t, err, "the restore tool did not start")
		assert.NotContains(t, err.Error(), "partially restored")
		assert.Empty(t, pgRestore.GetInvocations(t))
	})

	t.Run("feeds the restore tool the whole dump of an encrypted backup", func(t *testing.T) {
		t.Parallel()
		sink, pgRestore := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		restorePipeline := newTestPipeline(t, newDiscardingStorage())
		dump := []byte("PGDMP " + encryptionMarker)
		backup, encryptedFile := encryptTestBackup(t, restorePipeline, dump)

		err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
			backup, &memoryStorageReader{file: encryptedFile, failAtEndErr: nil}, sink, getStoredCompressionNone(),
		))

		require.NoError(t, err)
		assert.NotContains(t, string(encryptedFile), encryptionMarker)
		assert.Equal(t, dump, pgRestore.GetStdin(t))
	})

	t.Run("reports the backup stream failure when a byte of the encrypted file is changed", func(t *testing.T) {
		t.Parallel()
		sink, _ := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		restorePipeline := newTestPipeline(t, newDiscardingStorage())
		backup, encryptedFile := encryptTestBackup(t, restorePipeline, bytes.Repeat([]byte("PGDMP"), 1000))
		encryptedFile[len(encryptedFile)/2] ^= 0xFF

		err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
			backup, &memoryStorageReader{file: encryptedFile, failAtEndErr: nil}, sink, getStoredCompressionNone(),
		))

		require.Error(t, err)
		assert.Contains(t, err.Error(), backupStreamFailure)
	})

	t.Run("reports the backup stream failure when the encrypted file is cut inside a chunk", func(t *testing.T) {
		t.Parallel()
		sink, _ := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		restorePipeline := newTestPipeline(t, newDiscardingStorage())
		backup, encryptedFile := encryptTestBackup(t, restorePipeline, bytes.Repeat([]byte("PGDMP"), 1000))

		err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
			backup,
			&memoryStorageReader{file: encryptedFile[:len(encryptedFile)-2500], failAtEndErr: nil},
			sink,
			getStoredCompressionNone(),
		))

		require.Error(t, err)
		assert.Contains(t, err.Error(), backupStreamFailure)
	})

	t.Run("fails before the restore tool reads a byte when the file is another backup's", func(t *testing.T) {
		t.Parallel()
		sink, pgRestore := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		restorePipeline := newTestPipeline(t, newDiscardingStorage())
		backup, _ := encryptTestBackup(t, restorePipeline, []byte("PGDMP first backup"))
		_, otherBackupFile := encryptTestBackup(t, restorePipeline, []byte("PGDMP second backup"))

		err := restorePipeline.restoreFromStorage(t.Context(), newStubRestoreRun(
			backup, &memoryStorageReader{file: otherBackupFile, failAtEndErr: nil}, sink, getStoredCompressionNone(),
		))

		require.ErrorIs(t, err, crypto.ErrStreamHeaderMismatch)
		assert.Empty(t, pgRestore.GetInvocations(t))
		assert.Empty(t, pgRestore.GetStdin(t))
	})
}

func TestBackupRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("restores the dump an encrypted backup stored", func(t *testing.T) {
		t.Parallel()
		storage := &memoryStorage{}
		roundTripPipeline := newTestPipeline(t, storage)
		spec := newTestBackupSpec()
		spec.IsEncrypted = true

		backupResult, err := roundTripPipeline.streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  openStubPostgresqlDumpSource(t, "printf 'PGDMP "+encryptionMarker+"'"),
			compression: getStoredCompressionNone(),
		})
		require.NoError(t, err)
		sink, pgRestore := openStubPostgresqlRestoreSink(t, toolstub.Spec{IsReadingStdin: true, Script: ""})
		storageReader, err := storage.GetStorageReader(t.Context(), spec.StorageID)
		require.NoError(t, err)

		err = roundTripPipeline.restoreFromStorage(t.Context(), newStubRestoreRun(StoredBackup{
			BackupID:         spec.BackupID,
			StorageID:        spec.StorageID,
			FileName:         spec.FileName,
			EncryptionHeader: backupResult.EncryptionHeader,
		}, storageReader, sink, getStoredCompressionNone()))

		require.NoError(t, err)
		assert.Equal(t, []byte("PGDMP "+encryptionMarker), pgRestore.GetStdin(t))
	})
}

func newStubRestoreRun(
	backup StoredBackup,
	storageReader storages.StorageReader,
	sink logicalRestoreSink,
	compression storedCompression,
) restoreRun {
	return restoreRun{
		backup:        backup,
		storageReader: storageReader,
		sink:          sink,
		compression:   compression,
		logger:        slog.New(slog.DiscardHandler),
	}
}

func TestRestoreThatPanics(t *testing.T) {
	t.Parallel()

	t.Run("returns the panic as an error, so the restore gets an outcome", func(t *testing.T) {
		t.Parallel()

		openedTarget := &OpenedRestoreTarget{
			sink:        nil,
			compression: getStoredCompressionNone(),
			pipeline:    newTestPipeline(t, panickingStorage{}),
			logger:      slog.New(slog.DiscardHandler),
		}

		restoreErr := openedTarget.Restore(t.Context(), newUnencryptedStoredBackup())

		var panicError *background.PanicError
		require.ErrorAs(t, restoreErr, &panicError)
		assert.Equal(t, storageReaderPanic, panicError.Value)
	})
}

const storageReaderPanic = "the storage reader panicked"

// panickingStorage panics when a restore reads from it.
type panickingStorage struct{}

func (panickingStorage) StartBackgroundWrite(context.Context, storages.BackgroundWriteSpec) storages.BackgroundWrite {
	panic("the panicking storage writes nothing")
}

func (panickingStorage) GetStorageReader(context.Context, uuid.UUID) (storages.StorageReader, error) {
	panic(storageReaderPanic)
}

func newUnencryptedStoredBackup() StoredBackup {
	backupID := uuid.New()

	return StoredBackup{
		BackupID:         backupID,
		StorageID:        uuid.New(),
		FileName:         "test-backup-" + backupID.String(),
		EncryptionHeader: nil,
	}
}

// encryptTestBackup encrypts dump the way an encrypted backup stores it and returns the backup that
// names the stored file and its header.
func encryptTestBackup(t *testing.T, restorePipeline *Pipeline, dump []byte) (StoredBackup, []byte) {
	t.Helper()

	backup := newUnencryptedStoredBackup()

	var encryptedFile bytes.Buffer

	encryptingWriter, header, err := crypto.NewEncryptingWriter(
		&encryptedFile,
		restorePipeline.masterKey,
		backup.BackupID,
	)
	require.NoError(t, err)
	_, err = encryptingWriter.Write(dump)
	require.NoError(t, err)
	require.NoError(t, encryptingWriter.Close())

	backup.EncryptionHeader = &EncryptionHeader{
		Salt:  base64.StdEncoding.EncodeToString(header.Salt),
		Nonce: base64.StdEncoding.EncodeToString(header.Nonce),
	}

	return backup, encryptedFile.Bytes()
}

// encodeStoredFile returns dump in the stored format of an engine with the compression.
func encodeStoredFile(t *testing.T, compression storedCompression, dump []byte) []byte {
	t.Helper()

	if !compression.isZstd {
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

func openStubPostgresqlRestoreSink(t *testing.T, spec toolstub.Spec) (logicalRestoreSink, *toolstub.Stub) {
	t.Helper()

	clientTools := pgshared.ClientTools{Root: t.TempDir()}
	pgRestore := toolstub.Write(t, clientTools.GetExecutable(pgshared.Version16, pgshared.ExecutablePgRestore), spec)

	tunnel, err := pglogical.NewSource(pglogical.Settings{
		Host: "127.0.0.1", Port: 5432, Username: "restore_user", Password: "password", Database: "orders",
	}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &postgresqlRestoreSink{
		tunnel: tunnel,
		spec: pglogical.RestoreSpec{
			NewCleanupContext:   nil,
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

func openStubMysqlRestoreSink(t *testing.T, spec toolstub.Spec) (logicalRestoreSink, *toolstub.Stub) {
	t.Helper()

	clientToolsRoot := t.TempDir()
	binDir := filepath.Join(clientToolsRoot, "mysql", "mysql-"+string(mysql_engine.Version80), "bin")
	toolstub.Write(t, filepath.Join(binDir, string(mysql_engine.ExecutableMysqldump)), toolstub.Spec{})
	mysqlClient := toolstub.Write(t, filepath.Join(binDir, string(mysql_engine.ExecutableMysql)), spec)

	tunnel, err := mysql_engine.NewSource(mysql_engine.Settings{
		Host: "127.0.0.1", Port: 3306, Username: "restore_user", Password: "password", Database: "orders",
	}, mysql_engine.ClientTools{Root: clientToolsRoot}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mysqlRestoreSink{
		tunnel: tunnel,
		spec: mysql_engine.RestoreSpec{
			Version:            mysql_engine.Version80,
			SourceDatabaseName: "orders",
			Logger:             slog.New(slog.DiscardHandler),
		},
	}, mysqlClient
}

func openStubMariadbRestoreSink(t *testing.T, spec toolstub.Spec) (logicalRestoreSink, *toolstub.Stub) {
	t.Helper()

	clientToolsRoot := t.TempDir()
	clientVersion := mariadb_engine.GetClientVersionForServer(mariadb_engine.Version1011)
	binDir := filepath.Join(clientToolsRoot, "mariadb", "mariadb-"+string(clientVersion), "bin")
	toolstub.Write(t, filepath.Join(binDir, string(mariadb_engine.ExecutableMariadbDump)), toolstub.Spec{})
	mariadbClient := toolstub.Write(t, filepath.Join(binDir, string(mariadb_engine.ExecutableMariadb)), spec)

	tunnel, err := mariadb_engine.NewSource(mariadb_engine.Settings{
		Host: "127.0.0.1", Port: 3306, Username: "restore_user", Password: "password", Database: "orders",
	}, mariadb_engine.ClientTools{Root: clientToolsRoot}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mariadbRestoreSink{
		tunnel: tunnel,
		spec: mariadb_engine.RestoreSpec{
			Version:             mariadb_engine.Version1011,
			SourceDatabaseName:  "orders",
			IsSkipGaleraDisable: false,
			Logger:              slog.New(slog.DiscardHandler),
		},
	}, mariadbClient
}

func openStubMongodbRestoreSink(t *testing.T, spec toolstub.Spec) (logicalRestoreSink, *toolstub.Stub) {
	t.Helper()

	clientTools := mongodb_engine.ClientTools{Root: t.TempDir()}
	mongorestore := toolstub.Write(t, clientTools.GetExecutable(mongodb_engine.ExecutableMongorestore), spec)

	tunnel, err := mongodb_engine.NewSource(mongodb_engine.Settings{
		Host: "127.0.0.1", Port: 27017, Username: "restore_user", Password: "password", Database: "orders",
	}).OpenTunnel(t.Context(), slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	return &mongodbRestoreSink{
		tunnel: tunnel,
		spec: mongodb_engine.RestoreSpec{
			ClientTools:        clientTools,
			SourceDatabaseName: "orders",
			InsertionWorkers:   1,
			Logger:             slog.New(slog.DiscardHandler),
		},
	}, mongorestore
}
