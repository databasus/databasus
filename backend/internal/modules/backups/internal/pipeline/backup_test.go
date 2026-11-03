package pipeline

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/backups/internal/runcontext"
)

const (
	dumpToolError    = "error: query failed: ERROR:  permission denied for sequence orders_id_seq"
	encryptionMarker = "encryption-marker-7f3a"
)

// stubDumpEngine opens a dump source of one engine whose dump tool is a stub running dumpScript.
type stubDumpEngine struct {
	name           string
	compression    storedCompression
	openDumpSource func(t *testing.T, dumpScript string) logicalDumpSource
}

func getStubDumpEngines() []stubDumpEngine {
	return []stubDumpEngine{
		{
			name:           "PostgreSQL",
			compression:    (&postgresqlEngine{}).GetStoredCompression(),
			openDumpSource: openStubPostgresqlDumpSource,
		},
		{
			name:           "MySQL",
			compression:    (&mysqlEngine{}).GetStoredCompression(),
			openDumpSource: openStubMysqlDumpSource,
		},
		{
			name:           "MariaDB",
			compression:    (&mariadbEngine{}).GetStoredCompression(),
			openDumpSource: openStubMariadbDumpSource,
		},
		{
			name:           "MongoDB",
			compression:    (&mongodbEngine{}).GetStoredCompression(),
			openDumpSource: openStubMongodbDumpSource,
		},
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
				backupPipeline := newTestPipeline(t, newDiscardingStorage())

				_, err := backupPipeline.streamDumpToStorage(t.Context(), backupStream{
					spec:        newTestBackupSpec(),
					dumpSource:  dumpSource,
					compression: engine.compression,
				})

				require.Error(t, err)
				assert.Contains(t, err.Error(), dumpToolError)
				assert.NotContains(t, err.Error(), "save to storage")
			})

			t.Run("reports the storage error when the storage fails at once", func(t *testing.T) {
				t.Parallel()
				storageErr := errors.New("storage is unreachable")
				backupPipeline := newTestPipeline(
					t,
					&failingStorage{failureErr: storageErr, isFailingBeforeReading: true},
				)

				_, err := backupPipeline.streamDumpToStorage(t.Context(), backupStream{
					spec:        newTestBackupSpec(),
					dumpSource:  engine.openDumpSource(t, "head -c 4194304 /dev/zero"),
					compression: engine.compression,
				})

				assertStorageErrorReported(t, err, storageErr)
			})

			t.Run(
				"reports a cancelled backup when it is cancelled while the storage stopped reading",
				func(t *testing.T) {
					t.Parallel()
					runCtx, cancel := context.WithCancel(t.Context())
					t.Cleanup(cancel)
					storage := &stalledStorage{}
					backupPipeline := newTestPipeline(t, storage)
					dumpSource := engine.openDumpSource(t, "head -c 4194304 /dev/zero")
					backupErrors := make(chan error, 1)

					go func() {
						_, err := backupPipeline.streamDumpToStorage(runCtx, backupStream{
							spec:        newTestBackupSpec(),
							dumpSource:  dumpSource,
							compression: engine.compression,
						})
						backupErrors <- err
					}()

					testwait.Until(t, "the storage read the first byte of the dump", func(context.Context) error {
						if !storage.hasReadData.Load() {
							return errors.New("the storage has read nothing yet")
						}

						return nil
					})
					cancel()

					require.ErrorIs(t, <-backupErrors, runcontext.ErrBackupCancelled)
				},
			)
		})
	}

	t.Run("stores an encrypted backup without the dumped text", func(t *testing.T) {
		t.Parallel()
		storage := &memoryStorage{}
		backupPipeline := newTestPipeline(t, storage)
		spec := newTestBackupSpec()
		spec.IsEncrypted = true

		backupResult, err := backupPipeline.streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  openStubPostgresqlDumpSource(t, "printf '"+encryptionMarker+"'"),
			compression: getStoredCompressionNone(),
		})

		require.NoError(t, err)
		require.NotNil(t, backupResult.EncryptionHeader)
		assert.NotEmpty(t, storage.getStoredFile())
		assert.NotContains(t, string(storage.getStoredFile()), encryptionMarker)
	})

	t.Run("stores two encrypted backups of one dump as different files with different salts", func(t *testing.T) {
		t.Parallel()
		firstStorage := &memoryStorage{}
		secondStorage := &memoryStorage{}
		firstSpec := newTestBackupSpec()
		firstSpec.IsEncrypted = true
		secondSpec := newTestBackupSpec()
		secondSpec.IsEncrypted = true

		firstResult, err := newTestPipeline(t, firstStorage).streamDumpToStorage(t.Context(), backupStream{
			spec:        firstSpec,
			dumpSource:  openStubPostgresqlDumpSource(t, "printf 'unchanged dump'"),
			compression: getStoredCompressionNone(),
		})
		require.NoError(t, err)
		secondResult, err := newTestPipeline(t, secondStorage).streamDumpToStorage(t.Context(), backupStream{
			spec:        secondSpec,
			dumpSource:  openStubPostgresqlDumpSource(t, "printf 'unchanged dump'"),
			compression: getStoredCompressionNone(),
		})
		require.NoError(t, err)

		assert.NotEqual(t, firstStorage.getStoredFile(), secondStorage.getStoredFile())
		assert.NotEqual(t, firstResult.EncryptionHeader.Salt, secondResult.EncryptionHeader.Salt)
		assert.Equal(t, decodeBase64(t, firstResult.EncryptionHeader.Salt), firstStorage.getStoredFile()[8:40])
	})

	t.Run("reports the stored size of a MySQL dump after zstd when the dump completes", func(t *testing.T) {
		t.Parallel()
		var lastReportedMb float64
		spec := newTestBackupSpec()
		spec.OnProgress = func(storedSizeMb float64) { lastReportedMb = storedSizeMb }

		backupResult, err := newTestPipeline(t, newDiscardingStorage()).streamDumpToStorage(t.Context(), backupStream{
			spec:        spec,
			dumpSource:  openStubMysqlDumpSource(t, "head -c 3145728 /dev/zero"),
			compression: (&mysqlEngine{}).GetStoredCompression(),
		})

		require.NoError(t, err)
		assert.Positive(t, lastReportedMb)
		assert.Less(t, lastReportedMb, 1.0, "3 MB of zeros compress far below 1 MB")
		assert.InDelta(t, lastReportedMb, backupResult.StoredSizeMb, 0)
	})
}

func assertStorageErrorReported(t *testing.T, backupErr, storageErr error) {
	t.Helper()

	require.ErrorIs(t, backupErr, storageErr)
	assert.Contains(t, backupErr.Error(), "save to storage")
	assert.NotErrorIs(t, backupErr, runcontext.ErrBackupCancelled)
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

func TestRunBackupDuringSetup(t *testing.T) {
	t.Parallel()

	t.Run("returns the time limit error when the limit ends during setup", func(t *testing.T) {
		t.Parallel()

		runCtx, _, release := runcontext.New(t.Context(), runcontext.Limits{
			Shutdown:  runcontext.ErrBackupShutdown,
			Timeout:   runcontext.ErrBackupTimeout,
			TimeLimit: time.Nanosecond,
		})
		defer release()

		<-runCtx.Done()

		_, err := newTestPipeline(t, newDiscardingStorage()).RunBackup(runCtx, newTestBackupSpec())

		require.ErrorIs(t, err, runcontext.ErrBackupTimeout)
		assert.Equal(t, "backup exceeded the time limit of 23 hours", err.Error())
	})
}
