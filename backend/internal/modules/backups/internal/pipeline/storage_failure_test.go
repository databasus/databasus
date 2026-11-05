package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	test_utils "databasus-backend/internal/lib/httpclienttest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/databases"
)

// TestRunBackupWithFailingStorage dumps a real database of every engine into a storage that fails.
// It pins issue #816: the storage failure ends the backup context, and the backup was reported as
// cancelled instead of with the storage error.
func TestRunBackupWithFailingStorage(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	for _, engine := range []struct {
		name           string
		compression    storedCompression
		openDumpSource func(t *testing.T) logicalDumpSource
	}{
		{
			name:        "PostgreSQL",
			compression: getStoredCompressionNone(),
			openDumpSource: func(t *testing.T) logicalDumpSource {
				t.Helper()

				server := containers.StartPostgres(t, "postgres:16")
				dumpSource, err := openPostgresqlDumpSource(t.Context(), databases.PostgresqlLogicalConnectionSettings{
					Engine: pglogical.Settings{
						Host: server.Host, Port: server.Port, Username: containers.PostgresUsername,
						Password: containers.PostgresPassword, Database: containers.PostgresDatabase,
					},
					Version: pgshared.Version16,
				}, pgshared.ClientTools{Root: getClientToolsDir(t)}, logger)
				require.NoError(t, err)

				return dumpSource
			},
		},
		{
			name:        "MySQL",
			compression: (&mysqlEngine{}).GetStoredCompression(),
			openDumpSource: func(t *testing.T) logicalDumpSource {
				t.Helper()

				server := containers.StartMysql(t, "mysql:8.0")
				dumpSource, err := openMysqlDumpSource(t.Context(), databases.MysqlConnectionSettings{
					Engine: mysql_engine.Settings{
						Host: server.Host, Port: server.Port, Username: containers.MysqlUsername,
						Password: containers.MysqlPassword, Database: containers.MysqlDatabase,
					},
					Version: mysql_engine.Version80,
				}, mysql_engine.ClientTools{Root: getClientToolsDir(t)}, logger)
				require.NoError(t, err)

				return dumpSource
			},
		},
		{
			name:        "MariaDB",
			compression: (&mariadbEngine{}).GetStoredCompression(),
			openDumpSource: func(t *testing.T) logicalDumpSource {
				t.Helper()

				server := containers.StartMariadb(t, "mariadb:10.11")
				dumpSource, err := openMariadbDumpSource(t.Context(), databases.MariadbConnectionSettings{
					Engine: mariadb_engine.Settings{
						Host: server.Host, Port: server.Port, Username: containers.MariadbUsername,
						Password: containers.MariadbPassword, Database: containers.MariadbDatabase,
					},
					Version: mariadb_engine.Version1011,
				}, mariadb_engine.ClientTools{Root: getClientToolsDir(t)}, logger)
				require.NoError(t, err)

				return dumpSource
			},
		},
		{
			name:        "MongoDB",
			compression: getStoredCompressionNone(),
			openDumpSource: func(t *testing.T) logicalDumpSource {
				t.Helper()

				server := containers.StartMongodb(t, "mongo:7.0")
				dumpSource, err := openMongodbDumpSource(t.Context(), databases.MongodbConnectionSettings{
					Engine: mongodb_engine.Settings{
						Host: server.Host, Port: server.Port, Username: containers.MongodbUsername,
						Password: containers.MongodbPassword, Database: containers.MongodbDatabase,
						AuthDatabase: containers.MongodbAuthDatabase,
					},
					Version:  mongodb_engine.Version7,
					CPUCount: 1,
				}, mongodb_engine.ClientTools{Root: getClientToolsDir(t)}, logger)
				require.NoError(t, err)

				return dumpSource
			},
		},
	} {
		t.Run(engine.name, func(t *testing.T) {
			dumpSource := engine.openDumpSource(t)
			t.Cleanup(dumpSource.Close)
			storageErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))

			t.Run("reports the storage error when the storage times out after the dump finished", func(t *testing.T) {
				storage := &failingStorage{failureErr: storageErr, isFailingBeforeReading: false}

				_, err := newTestPipeline(t, storage).streamDumpToStorage(t.Context(), backupStream{
					spec:        newTestBackupSpec(),
					dumpSource:  dumpSource,
					compression: engine.compression,
				})

				require.Positive(t, storage.readBytes.Load(), "the dump never reached the storage")
				assertStorageErrorReported(t, err, storageErr)
			})

			t.Run("reports the storage error when the storage fails before reading", func(t *testing.T) {
				storage := &failingStorage{failureErr: storageErr, isFailingBeforeReading: true}

				_, err := newTestPipeline(t, storage).streamDumpToStorage(t.Context(), backupStream{
					spec:        newTestBackupSpec(),
					dumpSource:  dumpSource,
					compression: engine.compression,
				})

				assertStorageErrorReported(t, err, storageErr)
			})
		})
	}
}

func getClientToolsDir(t *testing.T) string {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)

	clientToolsDir, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	return clientToolsDir
}
