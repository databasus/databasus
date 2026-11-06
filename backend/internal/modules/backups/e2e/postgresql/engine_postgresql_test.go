package postgresql

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const (
	timescaledbImage = "timescale/timescaledb:2.17.0-pg17"

	// skippedTableMinSizeMb keeps a table left out of a dump large enough that the recorded raw size
	// visibly leaves it out.
	skippedTableMinSizeMb = 15.0
)

func getPostgresqlVersions() []struct {
	name  string
	tag   string
	image string
} {
	return []struct {
		name  string
		tag   string
		image string
	}{
		{"PostgreSQL 12", "12", "postgres:12"},
		{"PostgreSQL 13", "13", "postgres:13"},
		{"PostgreSQL 14", "14", "postgres:14"},
		{"PostgreSQL 15", "15", "postgres:15"},
		{"PostgreSQL 16", "16", "postgres:16"},
		{"PostgreSQL 17", "17", "postgres:17"},
		{"PostgreSQL 18", "18", "postgres:18"},
	}
}

type postgresqlTestRow struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

type relationRef struct {
	SchemaName string
	TableName  string
}

// TestBackupAndRestorePostgresql boots each supported version once and runs every scenario against
// it, so only one server of the matrix is alive at a time.
//
//nolint:tparallel // Each version's scenarios share one server, and a version stops before the next starts.
func TestBackupAndRestorePostgresql(t *testing.T) {
	t.Parallel()

	for _, postgresqlVersion := range getPostgresqlVersions() {
		t.Run(postgresqlVersion.name, func(t *testing.T) {
			endpoint := containers.StartPostgres(t, postgresqlVersion.image)

			t.Run("restores the rows with one job through a stream", func(t *testing.T) {
				testPostgresqlRowsRestoredWithCPUCount(t, endpoint, 1)
			})
			t.Run("restores the rows with four jobs through a directory", func(t *testing.T) {
				testPostgresqlRowsRestoredWithCPUCount(t, endpoint, 4)
			})
			t.Run("restores the rows of an encrypted backup", func(t *testing.T) {
				testPostgresqlEncryptedBackupRestored(t, endpoint)
			})
			t.Run("restores every schema when the database includes none", func(t *testing.T) {
				testPostgresqlAllSchemasRestored(t, endpoint)
			})
			t.Run("restores the rows when the restore excludes the extensions the target has", func(t *testing.T) {
				testPostgresqlRestoreExcludingExtensions(t, endpoint)
			})
			t.Run("recovers the extensions when the restore keeps them", func(t *testing.T) {
				testPostgresqlRestoreKeepingExtensions(t, endpoint)
			})
			t.Run("keeps the owner of a table when the restore keeps ownership", func(t *testing.T) {
				testPostgresqlRestoreKeepingOwnership(t, endpoint)
			})
			t.Run("keeps the grants of a table when the restore keeps privileges", func(t *testing.T) {
				testPostgresqlRestoreKeepingPrivileges(t, endpoint)
			})
			t.Run("restores only the included schemas and sizes the backup without the rest", func(t *testing.T) {
				testPostgresqlOnlyIncludedSchemasRestored(t, endpoint)
			})
			t.Run("leaves out an excluded table and sizes the backup without it", func(t *testing.T) {
				testPostgresqlExcludedTableLeftOut(t, endpoint)
			})
			t.Run("leaves out an excluded table of an included schema", func(t *testing.T) {
				testPostgresqlExcludedTableOfIncludedSchemaLeftOut(t, endpoint)
			})
			t.Run("restores a table without its rows when its data is excluded", func(t *testing.T) {
				testPostgresqlExcludedTableDataLeftOut(t, endpoint)
			})
			t.Run("matches the exclusion patterns the way pg_dump does", func(t *testing.T) {
				testPostgresqlExclusionPatterns(t, endpoint)
			})
			t.Run("empties only the partitions a data exclusion matches", func(t *testing.T) {
				testPostgresqlExcludedPartitionData(t, endpoint)
			})
			t.Run("excludes table data only inside the included schemas", func(t *testing.T) {
				testPostgresqlExcludedTableDataOfIncludedSchemas(t, endpoint)
			})
			t.Run("restores the rows of a backup taken by a limited user", func(t *testing.T) {
				testPostgresqlLimitedUserBackupRestored(t, endpoint)
			})
			t.Run("fails the backup when the user cannot read a sequence", func(t *testing.T) {
				testPostgresqlBackupFailsWithoutSequenceAccess(t, endpoint, postgresqlVersion.tag)
			})
			t.Run(
				"fails and cleans up only the failed attempt after rls restricts saved credentials",
				func(t *testing.T) {
					testPostgresqlBackupFailsAfterRLSDrift(t, endpoint)
				},
			)
			t.Run("restores the rows when the dump skips user mappings", func(t *testing.T) {
				testPostgresqlRestoreSkippingUserMappings(t, endpoint)
			})
		})
	}
}

//nolint:tparallel // The subtests share one source server.
func TestBackupAndRestoreTimescaledb(t *testing.T) {
	t.Parallel()

	endpoint := containers.StartTimescaleDB(t, timescaledbImage)

	t.Run("restores the hypertable with its chunks with one job", func(t *testing.T) {
		testTimescaledbHypertableRestored(t, endpoint, 1)
	})

	// Four jobs restore through a directory, the path that failed with _timescaledb_catalog foreign
	// key violations.
	t.Run("restores the hypertable with its chunks with four jobs", func(t *testing.T) {
		testTimescaledbHypertableRestored(t, endpoint, 4)
	})

	// A TimescaleDB restore runs without --clean --if-exists, so the CREATE SCHEMA public an archive
	// dumped with -n public carries collides with the target's schema. pg_restore exits non-zero on
	// that one ignored item, and the restore must still succeed (issue #726).
	t.Run("restores the rows when the backup includes the public schema the target has", func(t *testing.T) {
		server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
		tableName := "readings_" + enginetest.NewUniqueSuffix()

		server.execute(t, fmt.Sprintf(`
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE TABLE public.%[1]s (id INTEGER PRIMARY KEY, label TEXT NOT NULL);
INSERT INTO public.%[1]s (id, label) VALUES (1, 'alpha'), (2, 'beta'), (3, 'gamma');
`, tableName))
		server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS public.%s", tableName))

		settings := server.newSettings()
		settings.IncludeSchemas = []string{"public"}

		restored := server.backUpAndRestore(t, postgresqlCycle{
			settings:                settings,
			encryption:              apiclient.BackupEncryptionNONE,
			changeTarget:            nil,
			prepareRestoredDatabase: nil,
		})

		require.NotEmpty(t, restored.tm.GetTimescaledbVersion(t, restored.backup.Id),
			"the restore skips --clean only for a backup of a TimescaleDB database")
		assert.Equal(t, 3, countPostgresqlRows(t, restored.restoredDB,
			relationRef{SchemaName: "public", TableName: tableName}))
	})
}

func TestBackupAndRestorePostgresqlOverTls(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows over a connection that requires TLS", func(t *testing.T) {
		t.Parallel()

		endpoint := containers.StartPostgresSSL(t, enginetest.GetTestdataDirectory(t, "ssl"))
		server := connectToPostgresql(t, endpoint, "sslmode=require")
		tableName := "test_data_ssl_" + enginetest.NewUniqueSuffix()
		server.execute(t, createAndFillPostgresqlTableQuery(tableName))

		settings := server.newSettings()
		settings.SslMode = apiclient.PostgresqlSslModeRequire

		restored := server.backUpAndRestore(t, postgresqlCycle{
			settings:   settings,
			encryption: apiclient.BackupEncryptionNONE,
			changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
				target.SslMode = apiclient.PostgresqlSslModeRequire
			},
			prepareRestoredDatabase: nil,
		})

		verifyPostgresqlRows(t, server.admin, restored.restoredDB, tableName)
	})
}

//nolint:tparallel // The subtests share one source server.
func TestBackupAndRestorePostgresqlOverMutualTls(t *testing.T) {
	t.Parallel()

	endpoint := containers.StartPostgresMtls(t, enginetest.GetTestdataDirectory(t, "mtls"))
	clientCert, clientKey, rootCert := readMtlsCertificates(t)

	t.Run("restores the rows over a connection that requires a client certificate", func(t *testing.T) {
		server := connectToPostgresql(t, endpoint, newMtlsConnectionOptions(t, clientCert, clientKey, rootCert))
		tableName := "test_data_mtls_" + enginetest.NewUniqueSuffix()
		server.execute(t, createAndFillPostgresqlTableQuery(tableName))

		settings := server.newSettings()
		settings.SslMode = apiclient.PostgresqlSslModeVerifyCa
		settings.SslClientCert = &clientCert
		settings.SslClientKey = &clientKey
		settings.SslRootCert = &rootCert

		restored := server.backUpAndRestore(t, postgresqlCycle{
			settings:   settings,
			encryption: apiclient.BackupEncryptionNONE,
			changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
				target.SslMode = apiclient.PostgresqlSslModeVerifyCa
				target.SslClientCert = &clientCert
				target.SslClientKey = &clientKey
				target.SslRootCert = &rootCert
			},
			prepareRestoredDatabase: nil,
		})

		verifyPostgresqlRows(t, server.admin, restored.restoredDB, tableName)
	})

	t.Run("refuses the database without a client certificate", func(t *testing.T) {
		tm := enginetest.NewTeam(t)
		server := postgresqlServer{endpoint: endpoint, connectionOptions: "", admin: nil}

		settings := server.newSettings()
		settings.SslMode = apiclient.PostgresqlSslModeRequire

		response, err := tm.Owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.Workspace.Id,
			Name:        "Database " + enginetest.NewUniqueSuffix(),
			NotifierIds: []uuid.UUID{},
			Settings:    enginetest.NewDatabaseSettings(t, settings),
		})
		require.NoError(t, err)
		assert.NotEqual(t, http.StatusCreated, response.StatusCode(),
			"a database whose server requires a client certificate cannot be saved without one")
	})
}

func TestBackupAndRestorePostgresqlOverSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows through the bastion", func(t *testing.T) {
		t.Parallel()

		topology := containers.StartPostgresBehindSSHBastion(t, "postgres:16")
		server := connectToPostgresql(t, enginetest.OpenForwardedEndpoint(t, topology), postgresqlPlainConnection)
		tableName := "test_data_ssh_tunnel_" + enginetest.NewUniqueSuffix()
		server.execute(t, createAndFillPostgresqlTableQuery(tableName))

		bastionServer := postgresqlServer{endpoint: topology.Database, connectionOptions: "", admin: nil}
		settings := bastionServer.newSettings()
		settings.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, settings)
		backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName, restoredDB := server.createRestoreDatabase(t, "restored_ssh_tunnel")
		target := bastionServer.newTarget(restoredDatabaseName)
		target.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)
		tm.Restore(t, backup.Id, toPostgresqlRestoreTarget(t, target))

		verifyPostgresqlRows(t, server.admin, restoredDB, tableName)
	})
}

func TestBackupPostgresqlToFailingStorage(t *testing.T) {
	t.Parallel()

	t.Run("fails the backup without hanging when the upload fails", func(t *testing.T) {
		t.Parallel()

		server := connectToPostgresql(t, containers.StartPostgres(t, "postgres:16"), postgresqlPlainConnection)
		tableName := "test_data_" + enginetest.NewUniqueSuffix()
		server.execute(t, createAndFillPostgresqlTableQuery(tableName))

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, server.newSettings())

		tm.RequireBackupFailsWithoutHanging(t, database.Id)
	})
}

func testPostgresqlRowsRestoredWithCPUCount(t *testing.T, endpoint containers.Endpoint, cpuCount int) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	tableName := server.createFilledTable(t)

	settings := server.newSettings()
	settings.CpuCount = cpuCount

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   settings,
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.CpuCount = cpuCount
		},
		prepareRestoredDatabase: nil,
	})

	restored.tm.RequireBackupSizeMatchesDownload(t, restored.backup)
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: tableName}))
	verifyPostgresqlRows(t, server.admin, restored.restoredDB, tableName)
}

func testPostgresqlEncryptedBackupRestored(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	tableName := server.createFilledTable(t)

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                server.newSettings(),
		encryption:              apiclient.BackupEncryptionENCRYPTED,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.Equal(t, apiclient.BackupEncryptionENCRYPTED, restored.backup.Encryption)
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: tableName}))
	verifyPostgresqlRows(t, server.admin, restored.restoredDB, tableName)
}

func testPostgresqlAllSchemasRestored(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	server.createThreeSchemas(t, "")

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                server.newSettings(),
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "public_table"}), "public.public_table is restored")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_a", TableName: "table_a"}), "schema_a.table_a is restored")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_b", TableName: "table_b"}), "schema_b.table_b is restored")
}

func testPostgresqlRestoreExcludingExtensions(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)

	// The comment on the extension checks that COMMENT ON EXTENSION is left out along with it.
	server.execute(t, `
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		COMMENT ON EXTENSION "uuid-ossp" IS 'Test comment on uuid-ossp extension';
		DROP TABLE IF EXISTS test_extension_data;
		CREATE TABLE test_extension_data (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO test_extension_data (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	server.cleanUp(t, `DROP TABLE IF EXISTS test_extension_data; DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;`)

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   server.newSettings(),
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.IsExcludeExtensions = true
		},
		prepareRestoredDatabase: func(t *testing.T, restoredDB *sqlx.DB) {
			t.Helper()

			// A managed service installs its extensions into the target before the restore.
			_, err := restoredDB.ExecContext(t.Context(), `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
			require.NoError(t, err)
		},
	})

	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "test_extension_data"}))
	assert.Equal(t, 3, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "test_extension_data"}))
	assert.NotEmpty(t, generatePostgresqlUUID(t, restored.restoredDB), "uuid_generate_v4 works")
}

func testPostgresqlRestoreKeepingExtensions(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	server.execute(t, `
		DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		DROP TABLE IF EXISTS test_extension_recovery;
		CREATE TABLE test_extension_recovery (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO test_extension_recovery (name) VALUES ('test1'), ('test2'), ('test3');
	`)
	server.cleanUp(t, `DROP TABLE IF EXISTS test_extension_recovery; DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;`)

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   server.newSettings(),
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.IsExcludeExtensions = false
		},
		prepareRestoredDatabase: func(t *testing.T, restoredDB *sqlx.DB) {
			t.Helper()

			assert.False(t, isPostgresqlExtensionInstalled(t, restoredDB, "uuid-ossp"),
				"the extension is missing from the target before the restore")
		},
	})

	assert.True(t, isPostgresqlExtensionInstalled(t, restored.restoredDB, "uuid-ossp"),
		"the restore recovers the extension")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "test_extension_recovery"}))
	assert.Equal(t, 3, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "test_extension_recovery"}))
	assert.NotEmpty(t, generatePostgresqlUUID(t, restored.restoredDB),
		"uuid_generate_v4 works after the extension is recovered")
}

func testPostgresqlRestoreKeepingOwnership(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	ownerRole := "test_owner_" + suffix
	tableName := "test_ownership_data_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE ROLE %[1]s LOGIN PASSWORD 'placeholder';
		CREATE TABLE %[2]s (id SERIAL PRIMARY KEY, name TEXT NOT NULL, value INTEGER NOT NULL);
		ALTER TABLE %[2]s OWNER TO %[1]s;
		INSERT INTO %[2]s (name, value) VALUES ('a', 1), ('b', 2), ('c', 3);
	`, ownerRole, tableName))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS %s; DROP ROLE IF EXISTS %s;", tableName, ownerRole))

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   server.newSettings(),
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.IsRestoreOwnership = true
			target.IsRestorePrivileges = false
		},
		prepareRestoredDatabase: nil,
	})

	var restoredOwner string
	require.NoError(t, restored.restoredDB.GetContext(t.Context(), &restoredOwner,
		"SELECT tableowner FROM pg_tables WHERE schemaname = 'public' AND tablename = $1", tableName))
	assert.Equal(t, ownerRole, restoredOwner, "the restored table keeps its original owner")
}

func testPostgresqlRestoreKeepingPrivileges(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	granteeRole := "test_grantee_" + suffix
	tableName := "test_privileges_data_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE ROLE %[1]s LOGIN PASSWORD 'placeholder';
		CREATE TABLE %[2]s (id SERIAL PRIMARY KEY, name TEXT NOT NULL, value INTEGER NOT NULL);
		INSERT INTO %[2]s (name, value) VALUES ('a', 1), ('b', 2), ('c', 3);
		GRANT SELECT, INSERT ON %[2]s TO %[1]s;
	`, granteeRole, tableName))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS %s; DROP ROLE IF EXISTS %s;", tableName, granteeRole))

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   server.newSettings(),
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.IsRestoreOwnership = false
			target.IsRestorePrivileges = true
		},
		prepareRestoredDatabase: nil,
	})

	var grantedPrivileges []string
	require.NoError(t, restored.restoredDB.SelectContext(t.Context(), &grantedPrivileges, `
		SELECT privilege_type FROM information_schema.role_table_grants
		WHERE grantee = $1 AND table_schema = 'public' AND table_name = $2
		ORDER BY privilege_type
	`, granteeRole, tableName))
	assert.Contains(t, grantedPrivileges, "SELECT", "the restored table keeps its SELECT grant")
	assert.Contains(t, grantedPrivileges, "INSERT", "the restored table keeps its INSERT grant")
}

func testPostgresqlOnlyIncludedSchemasRestored(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	server.createThreeSchemas(t,
		"INSERT INTO schema_b.table_b (data) SELECT repeat('x', 1024) FROM generate_series(1, 20000);")

	settings := server.newSettings()
	settings.IncludeSchemas = []string{"public", "schema_a"}

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                settings,
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	server.requireRawSizeLeavesOut(t, restored.backup, relationRef{SchemaName: "schema_b", TableName: "table_b"},
		"the recorded raw size must not count the schemas left out of the dump")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "public_table"}), "public.public_table was included")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_a", TableName: "table_a"}), "schema_a.table_a was included")
	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_b", TableName: "table_b"}), "schema_b.table_b was left out")
}

func testPostgresqlExcludedTableLeftOut(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	keptTable := "keep_me_" + suffix
	droppedTable := "drop_me_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE TABLE public.%[1]s (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE public.%[2]s (id SERIAL PRIMARY KEY, data TEXT);
		INSERT INTO public.%[1]s (data) VALUES ('keep_value');
		INSERT INTO public.%[2]s (data) VALUES ('drop_value');
		INSERT INTO public.%[2]s (data) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
	`, keptTable, droppedTable))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS public.%s, public.%s", keptTable, droppedTable))

	settings := server.newSettings()
	settings.ExcludeTables = []string{droppedTable}

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                settings,
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	server.requireRawSizeLeavesOut(t, restored.backup, relationRef{SchemaName: "public", TableName: droppedTable},
		"the recorded raw size must not count the excluded table")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: keptTable}), "the kept table is restored")
	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: droppedTable}), "the excluded table is left out")
}

func testPostgresqlExcludedTableOfIncludedSchemaLeftOut(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	server.createThreeSchemas(t, `
		CREATE TABLE schema_a.skip_me (id SERIAL PRIMARY KEY, data TEXT);
		INSERT INTO schema_a.skip_me (data) VALUES ('skipped_data');
	`)

	settings := server.newSettings()
	settings.IncludeSchemas = []string{"public", "schema_a"}
	settings.ExcludeTables = []string{"schema_a.skip_me"}

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                settings,
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: "public_table"}), "public.public_table: schema included")
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_a", TableName: "table_a"}), "schema_a.table_a: schema included, table kept")
	assert.False(t, isPostgresqlTableRestored(
		t,
		restored.restoredDB,
		relationRef{
			SchemaName: "schema_a",
			TableName:  "skip_me",
		},
	), "schema_a.skip_me: excluded inside an included schema")
	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "schema_b", TableName: "table_b"}), "schema_b.table_b: schema not included")
}

func testPostgresqlExcludedTableDataLeftOut(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	emptiedTable := relationRef{SchemaName: "public", TableName: "logs_" + suffix}
	neighbourTable := relationRef{SchemaName: "public", TableName: "orders_" + suffix}

	server.execute(t, fmt.Sprintf(`
		CREATE TABLE public.%[1]s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		CREATE INDEX %[1]s_payload_idx ON public.%[1]s (payload);
		INSERT INTO public.%[1]s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
		CREATE TABLE public.%[2]s (id SERIAL PRIMARY KEY, data TEXT NOT NULL);
		INSERT INTO public.%[2]s (data) VALUES ('first'), ('second');
	`, emptiedTable.TableName, neighbourTable.TableName))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS public.%s, public.%s",
		emptiedTable.TableName, neighbourTable.TableName))

	sourceIndexNames := getPostgresqlIndexNames(t, server.admin, emptiedTable)
	require.Len(t, sourceIndexNames, 2)

	sourceColumnDefinitions := getPostgresqlColumnDefinitions(t, server.admin, emptiedTable)

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                server.newScopedSettings(nil, nil, []string{"public." + emptiedTable.TableName}),
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB, emptiedTable))
	assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB, emptiedTable))
	assert.Equal(t, sourceIndexNames, getPostgresqlIndexNames(t, restored.restoredDB, emptiedTable))
	assert.Equal(t, sourceColumnDefinitions, getPostgresqlColumnDefinitions(t, restored.restoredDB, emptiedTable))
	assert.True(t, hasPostgresqlPrimaryKey(t, restored.restoredDB, emptiedTable))
	assert.Equal(t, 2, countPostgresqlRows(t, restored.restoredDB, neighbourTable))
	server.requireRawSizeLeavesOut(t, restored.backup, emptiedTable,
		"the recorded raw size must not count the table whose data is excluded")
}

func testPostgresqlExclusionPatterns(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	archiveSchema := "archive_" + suffix
	historyPrefix := "history_" + suffix
	logsTable := "logs_" + suffix
	auditTable := "audit_tmp_" + suffix
	overlapTable := "overlap_" + suffix
	oldOrdersTable := "old_orders_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE SCHEMA %[1]s;
		CREATE TABLE public.%[2]s_2024 (id INT);
		CREATE TABLE %[1]s.%[2]s_2025 (id INT);
		CREATE TABLE public.%[3]s (id INT);
		CREATE TABLE %[1]s.%[3]s (id INT);
		CREATE TABLE public.%[4]s (id INT);
		CREATE TABLE %[1]s.%[4]s (id INT);
		CREATE TABLE public.%[5]s (id INT);
		INSERT INTO public.%[2]s_2024 VALUES (1);
		INSERT INTO %[1]s.%[2]s_2025 VALUES (1);
		INSERT INTO public.%[3]s VALUES (1);
		INSERT INTO %[1]s.%[3]s VALUES (1), (2);
		INSERT INTO public.%[4]s VALUES (1);
		INSERT INTO %[1]s.%[4]s VALUES (1), (2);
		INSERT INTO public.%[5]s VALUES (1);
		CREATE TABLE public.%[6]s (id INT);
		CREATE TABLE %[1]s.%[6]s (id INT);
		INSERT INTO public.%[6]s VALUES (1);
		INSERT INTO %[1]s.%[6]s VALUES (1), (2);
	`, archiveSchema, historyPrefix, logsTable, auditTable, overlapTable, oldOrdersTable))
	server.cleanUp(t, fmt.Sprintf(
		"DROP SCHEMA IF EXISTS %s CASCADE; DROP TABLE IF EXISTS public.%s_2024, public.%s, public.%s, public.%s, public.%s",
		archiveSchema,
		historyPrefix,
		logsTable,
		auditTable,
		overlapTable,
		oldOrdersTable,
	))

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings: server.newScopedSettings(
			nil,
			[]string{auditTable, "public." + overlapTable, "public.old_orders_" + suffix[:4] + "*"},
			[]string{"*." + historyPrefix + "_*", logsTable, "public." + overlapTable},
		),
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	publicHistoryTable := relationRef{SchemaName: "public", TableName: historyPrefix + "_2024"}
	archiveHistoryTable := relationRef{SchemaName: archiveSchema, TableName: historyPrefix + "_2025"}
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB, publicHistoryTable))
	assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB, publicHistoryTable))
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB, archiveHistoryTable))
	assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB, archiveHistoryTable))

	assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: logsTable}))
	assert.Equal(t, 2, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: archiveSchema, TableName: logsTable}),
		"an unqualified data exclusion must not reach a table off the default search path")

	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: auditTable}))
	assert.Equal(t, 2, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: archiveSchema, TableName: auditTable}),
		"an unqualified whole-table exclusion must not reach a table off the default search path")

	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: oldOrdersTable}))
	assert.Equal(t, 2, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: archiveSchema, TableName: oldOrdersTable}),
		"a schema-qualified whole-table pattern must not reach another schema")

	assert.False(t, isPostgresqlTableRestored(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: overlapTable}),
		"a table in both lists must be excluded entirely")
}

func testPostgresqlExcludedPartitionData(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	parentNamedAlone := "events_a_" + suffix
	partitionsByPattern := "events_b_" + suffix

	createPartitionedTable := func(parentName string) string {
		return fmt.Sprintf(`
			CREATE TABLE public.%[1]s (id INT NOT NULL) PARTITION BY RANGE (id);
			CREATE TABLE public.%[1]s_2025 PARTITION OF public.%[1]s FOR VALUES FROM (0) TO (100);
			CREATE TABLE public.%[1]s_2026 PARTITION OF public.%[1]s FOR VALUES FROM (100) TO (200);
			INSERT INTO public.%[1]s VALUES (1), (2), (101);
		`, parentName)
	}

	server.execute(t, createPartitionedTable(parentNamedAlone)+createPartitionedTable(partitionsByPattern))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS public.%s, public.%s", parentNamedAlone, partitionsByPattern))

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings: server.newScopedSettings(nil, nil, []string{
			"public." + parentNamedAlone,
			"public." + partitionsByPattern + "_*",
		}),
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.Equal(t, 2, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: parentNamedAlone + "_2025"}),
		"naming only the partitioned parent must keep its partitions' rows")
	assert.Equal(t, 1, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: parentNamedAlone + "_2026"}))

	for _, tableName := range []string{
		partitionsByPattern,
		partitionsByPattern + "_2025",
		partitionsByPattern + "_2026",
	} {
		relation := relationRef{SchemaName: "public", TableName: tableName}

		assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB, relation), tableName)
		assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB, relation), tableName)
	}
}

func testPostgresqlExcludedTableDataOfIncludedSchemas(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	reportingSchema := "reporting_" + suffix
	logsTable := "logs_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE SCHEMA %[1]s;
		CREATE TABLE public.%[2]s (id INT);
		CREATE TABLE %[1]s.%[2]s (id INT);
		INSERT INTO public.%[2]s VALUES (1);
		INSERT INTO %[1]s.%[2]s VALUES (1);
	`, reportingSchema, logsTable))
	server.cleanUp(t, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE; DROP TABLE IF EXISTS public.%s",
		reportingSchema, logsTable))

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                server.newScopedSettings([]string{"public"}, nil, []string{logsTable}),
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	publicLogsTable := relationRef{SchemaName: "public", TableName: logsTable}
	assert.True(t, isPostgresqlTableRestored(t, restored.restoredDB, publicLogsTable))
	assert.Equal(t, 0, countPostgresqlRows(t, restored.restoredDB, publicLogsTable))
	assert.False(t, isPostgresqlSchemaRestored(t, restored.restoredDB, reportingSchema))
}

func testPostgresqlLimitedUserBackupRestored(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	tableName := server.createFilledTable(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, server.newSettings())

	limitedUser := tm.CreateLimitedUser(t, database)
	assert.NotEmpty(t, limitedUser.Username)
	assert.NotEmpty(t, limitedUser.Password)

	limitedDatabase := tm.UpdateDatabaseSettings(t, database,
		withPostgresqlCredentials(t, database, limitedUser.Username, limitedUser.Password))
	backup := tm.BackUp(t, limitedDatabase.Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.createRestoreDatabase(t, "restored_readonly")
	tm.Restore(t, backup.Id, toPostgresqlRestoreTarget(t, server.newTarget(restoredDatabaseName)))

	assert.True(t, isPostgresqlTableRestored(t, restoredDB, relationRef{SchemaName: "public", TableName: tableName}))
	verifyPostgresqlRows(t, server.admin, restoredDB, tableName)
}

// pg_dump 18.0 and 18.1 emitted wrong sequence values instead of failing when the dumping role could
// not read a sequence, so a backup completed while losing every sequence position (issue #725,
// fixed upstream in PostgreSQL 18.2).
func testPostgresqlBackupFailsWithoutSequenceAccess(t *testing.T, endpoint containers.Endpoint, postgresqlTag string) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	tableName := server.createFilledTable(t)
	sequenceName := tableName + "_id_seq"
	restrictedUsername := "no_sequence_reader_" + enginetest.NewUniqueSuffix()
	restrictedPassword := uuid.NewString()

	server.execute(t, fmt.Sprintf(`
		CREATE ROLE %[1]s LOGIN PASSWORD '%[2]s';
		GRANT CONNECT ON DATABASE %[3]s TO %[1]s;
		GRANT USAGE ON SCHEMA public TO %[1]s;
		GRANT SELECT ON ALL TABLES IN SCHEMA public TO %[1]s;
		GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO %[1]s;
	`, restrictedUsername, restrictedPassword, containers.PostgresDatabase))
	server.cleanUp(t, fmt.Sprintf("REASSIGN OWNED BY %[1]s TO %[2]s; DROP OWNED BY %[1]s; DROP ROLE IF EXISTS %[1]s;",
		restrictedUsername, containers.PostgresUsername))

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, server.newSettings())
	restrictedDatabase := tm.UpdateDatabaseSettings(t, database,
		withPostgresqlCredentials(t, database, restrictedUsername, restrictedPassword))

	// The sequence is revoked only after the database is saved: privileges drift on a live server, and
	// a backup must then fail loudly instead of writing a partial dump.
	server.execute(t, fmt.Sprintf("REVOKE ALL ON SEQUENCE %s FROM PUBLIC, %s", sequenceName, restrictedUsername))

	tm.EnableBackups(t, restrictedDatabase.Id, tm.StorageID, apiclient.BackupEncryptionNONE)
	failedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, restrictedDatabase.Id).Id)

	assert.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status,
		"PostgreSQL %s: the backup must fail when the dumping role cannot read a sequence", postgresqlTag)

	// PostgreSQL 18 says "failed to get data for sequence ..." where earlier majors say "permission
	// denied for sequence ..."; both name the sequence.
	if assert.NotNil(t, failedBackup.FailMessage) {
		assert.Contains(t, *failedBackup.FailMessage, sequenceName)
	}
}

// postgres_fdw stands in for oracle_fdw, which CI cannot install. The backup runs as a role that
// cannot read the options of a user mapping, so pg_dump emits a bare CREATE USER MAPPING unless the
// database skips user mappings.
func testPostgresqlRestoreSkippingUserMappings(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	suffix := enginetest.NewUniqueSuffix()
	limitedUsername := "um_backup_" + suffix
	limitedPassword := "limitedpassword123"
	serverName := "um_srv_" + suffix
	tableName := "skip_um_data_" + suffix

	server.execute(t, fmt.Sprintf(`
		CREATE TABLE %[4]s (id INT PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO %[4]s (id, name) VALUES (1, 'row1'), (2, 'row2'), (3, 'row3');
		CREATE EXTENSION IF NOT EXISTS postgres_fdw;
		CREATE SERVER %[1]s FOREIGN DATA WRAPPER postgres_fdw OPTIONS (host 'localhost', dbname 'postgres');
		CREATE USER MAPPING FOR CURRENT_USER SERVER %[1]s OPTIONS ("user" 'remote', password 'secret');
		CREATE USER "%[2]s" WITH PASSWORD '%[3]s' LOGIN;
		GRANT CONNECT ON DATABASE "%[5]s" TO "%[2]s";
		GRANT USAGE ON SCHEMA public TO "%[2]s";
		GRANT SELECT ON ALL TABLES IN SCHEMA public TO "%[2]s";
		GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO "%[2]s";
	`, serverName, limitedUsername, limitedPassword, tableName, containers.PostgresDatabase))
	server.cleanUp(t, fmt.Sprintf(`DROP SERVER IF EXISTS %[1]s CASCADE; DROP TABLE IF EXISTS %[3]s CASCADE;
		DROP OWNED BY "%[2]s"; DROP USER IF EXISTS "%[2]s";`, serverName, limitedUsername, tableName))

	// The database saved under the limited role with user mappings skipped passes the connection
	// test's check of unreadable user mappings, and the backup runs as that role.
	settings := server.newSettings()
	settings.Username = limitedUsername
	settings.Password = &limitedPassword
	settings.IsSkipUserMappings = true

	// The restore connects as the admin so the dumped extension and server can be created; the
	// database's saved flag leaves the user mapping out.
	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:                settings,
		encryption:              apiclient.BackupEncryptionNONE,
		changeTarget:            nil,
		prepareRestoredDatabase: nil,
	})

	assert.Equal(t, 3, countPostgresqlRows(t, restored.restoredDB,
		relationRef{SchemaName: "public", TableName: tableName}), "the table rows are restored")

	var userMappingCount int
	require.NoError(t, restored.restoredDB.GetContext(t.Context(), &userMappingCount, `
		SELECT COUNT(*) FROM pg_user_mappings um
		JOIN pg_foreign_server s ON s.oid = um.srvid
		WHERE s.srvname = $1`, serverName))
	assert.Equal(t, 0, userMappingCount, "the user mapping is left out of the restore")
}

func testTimescaledbHypertableRestored(t *testing.T, endpoint containers.Endpoint, cpuCount int) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	tableName := "sensor_data_" + enginetest.NewUniqueSuffix()

	server.execute(t, fmt.Sprintf(`
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE TABLE %[1]s (
    time        TIMESTAMPTZ NOT NULL,
    sensor_id   INTEGER NOT NULL,
    temperature DOUBLE PRECISION NOT NULL
);
SELECT create_hypertable('%[1]s', 'time');
INSERT INTO %[1]s (time, sensor_id, temperature)
SELECT ts, (random() * 10)::int, random() * 100
FROM generate_series('2024-01-01'::timestamptz, '2024-03-01'::timestamptz, interval '1 hour') AS ts;
`, tableName))
	server.cleanUp(t, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))

	hypertable := relationRef{SchemaName: "public", TableName: tableName}
	sourceRowCount := countPostgresqlRows(t, server.admin, hypertable)
	require.Positive(t, sourceRowCount)

	settings := server.newSettings()
	settings.CpuCount = cpuCount

	restored := server.backUpAndRestore(t, postgresqlCycle{
		settings:   settings,
		encryption: apiclient.BackupEncryptionNONE,
		changeTarget: func(target *apiclient.PostgresqlRestoreTarget) {
			target.CpuCount = cpuCount
		},
		prepareRestoredDatabase: nil,
	})

	assert.NotEmpty(t, restored.tm.GetTimescaledbVersion(t, restored.backup.Id),
		"the backup records the TimescaleDB version of the source")
	assert.Equal(t, sourceRowCount, countPostgresqlRows(t, restored.restoredDB, hypertable),
		"the restored hypertable keeps every row")

	var hypertableCount int
	require.NoError(t, restored.restoredDB.GetContext(t.Context(), &hypertableCount,
		"SELECT count(*) FROM timescaledb_information.hypertables WHERE hypertable_name = $1", tableName))
	assert.Equal(t, 1, hypertableCount, "the restored table is a hypertable")

	var chunkCount int
	require.NoError(t, restored.restoredDB.GetContext(t.Context(), &chunkCount,
		"SELECT count(*) FROM timescaledb_information.chunks WHERE hypertable_name = $1", tableName))
	assert.Greater(t, chunkCount, 1, "the restored hypertable keeps its chunks")

	var targetDatabaseName string
	require.NoError(t, restored.restoredDB.GetContext(t.Context(), &targetDatabaseName, "SELECT current_database()"))
	assert.False(t, isTimescaleTargetRestoring(t, server, targetDatabaseName))
}

const postgresqlPlainConnection = "sslmode=disable"

// postgresqlServer is a PostgreSQL server of a test, reached with the container's admin user.
type postgresqlServer struct {
	endpoint containers.Endpoint
	// connectionOptions are the libpq options of the test's own connections, such as the sslmode.
	connectionOptions string
	admin             *sqlx.DB
}

func connectToPostgresql(t *testing.T, endpoint containers.Endpoint, connectionOptions string) postgresqlServer {
	t.Helper()

	server := postgresqlServer{endpoint: endpoint, connectionOptions: connectionOptions, admin: nil}
	server.admin = server.open(t, containers.PostgresDatabase)

	return server
}

func (s postgresqlServer) open(t *testing.T, databaseName string) *sqlx.DB {
	t.Helper()

	connection, err := sqlx.ConnectContext(t.Context(), "postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s %s",
		s.endpoint.Host, s.endpoint.Port, containers.PostgresUsername, containers.PostgresPassword,
		databaseName, s.connectionOptions,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })

	return connection
}

func (s postgresqlServer) execute(t *testing.T, statements string) {
	t.Helper()

	_, err := s.admin.ExecContext(t.Context(), statements)
	require.NoError(t, err)
}

// cleanUp runs the statements on the shared server when the test ends, so the next scenario starts
// from the same database.
func (s postgresqlServer) cleanUp(t *testing.T, statements string) {
	t.Helper()

	admin := s.admin

	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.WithoutCancel(t.Context()), statements)
	})
}

// createRestoreDatabase creates an empty database the test restores into and drops it when the
// test ends.
func (s postgresqlServer) createRestoreDatabase(t *testing.T, prefix string) (string, *sqlx.DB) {
	t.Helper()

	databaseName := prefix + "_" + enginetest.NewUniqueSuffix()
	s.execute(t, "CREATE DATABASE "+databaseName)
	s.cleanUp(t, "DROP DATABASE IF EXISTS "+databaseName)

	return databaseName, s.open(t, databaseName)
}

func (s postgresqlServer) createFilledTable(t *testing.T) string {
	t.Helper()

	tableName := "test_data_" + enginetest.NewUniqueSuffix()
	s.execute(t, createAndFillPostgresqlTableQuery(tableName))
	s.cleanUp(t, "DROP TABLE IF EXISTS "+tableName)

	return tableName
}

// createThreeSchemas creates public.public_table, schema_a.table_a and schema_b.table_b with a row
// each, runs the extra statements and drops them all when the test ends.
func (s postgresqlServer) createThreeSchemas(t *testing.T, extraStatements string) {
	t.Helper()

	s.execute(t, `
		DROP TABLE IF EXISTS public.public_table;
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
		CREATE SCHEMA schema_a;
		CREATE SCHEMA schema_b;
		CREATE TABLE public.public_table (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_a.table_a (id SERIAL PRIMARY KEY, data TEXT);
		CREATE TABLE schema_b.table_b (id SERIAL PRIMARY KEY, data TEXT);
		INSERT INTO public.public_table (data) VALUES ('public_data');
		INSERT INTO schema_a.table_a (data) VALUES ('schema_a_data');
		INSERT INTO schema_b.table_b (data) VALUES ('schema_b_data');
	`+extraStatements)
	s.cleanUp(t, `
		DROP TABLE IF EXISTS public.public_table;
		DROP SCHEMA IF EXISTS schema_a CASCADE;
		DROP SCHEMA IF EXISTS schema_b CASCADE;
	`)
}

func (s postgresqlServer) newSettings() apiclient.PostgresqlLogicalSettings {
	password := containers.PostgresPassword
	database := containers.PostgresDatabase

	return apiclient.PostgresqlLogicalSettings{
		Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:               s.endpoint.Host,
		Port:               s.endpoint.Port,
		Username:           containers.PostgresUsername,
		Password:           &password,
		Database:           &database,
		SslMode:            apiclient.PostgresqlSslModeDisable,
		SslClientCert:      nil,
		SslClientKey:       nil,
		SslRootCert:        nil,
		SshTunnel:          enginetest.NewDisabledSSHTunnelSettings(),
		IncludeSchemas:     []string{},
		ExcludeTables:      []string{},
		ExcludeTableData:   []string{},
		IsSkipUserMappings: false,
		CpuCount:           1,
		Version:            nil,
	}
}

func (s postgresqlServer) newScopedSettings(
	includeSchemas []string,
	excludeTables []string,
	excludeTableData []string,
) apiclient.PostgresqlLogicalSettings {
	settings := s.newSettings()
	settings.IncludeSchemas = getNonNilList(includeSchemas)
	settings.ExcludeTables = getNonNilList(excludeTables)
	settings.ExcludeTableData = getNonNilList(excludeTableData)

	return settings
}

func (s postgresqlServer) newTarget(databaseName string) apiclient.PostgresqlRestoreTarget {
	password := containers.PostgresPassword

	return apiclient.PostgresqlRestoreTarget{
		Type:                apiclient.PostgresqlRestoreTargetTypePOSTGRESLOGICAL,
		Host:                s.endpoint.Host,
		Port:                s.endpoint.Port,
		Username:            containers.PostgresUsername,
		Password:            &password,
		Database:            databaseName,
		SslMode:             apiclient.PostgresqlSslModeDisable,
		SslClientCert:       nil,
		SslClientKey:        nil,
		SslRootCert:         nil,
		SshTunnel:           enginetest.NewDisabledSSHTunnelSettings(),
		CpuCount:            1,
		IsExcludeExtensions: false,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
	}
}

// requireRawSizeLeavesOut checks that the raw size the backup recorded leaves out the relation the
// dump skipped, which is large enough to show.
func (s postgresqlServer) requireRawSizeLeavesOut(
	t *testing.T,
	backup apiclient.LogicalBackup,
	skippedRelation relationRef,
	message string,
) {
	t.Helper()

	var skippedSizeMb float64
	require.NoError(t, s.admin.GetContext(t.Context(), &skippedSizeMb,
		"SELECT pg_total_relation_size(format('%I.%I', $1::text, $2::text)::regclass) / (1024.0 * 1024.0)",
		skippedRelation.SchemaName, skippedRelation.TableName))

	var wholeDatabaseSizeMb float64
	require.NoError(t, s.admin.GetContext(t.Context(), &wholeDatabaseSizeMb,
		"SELECT pg_database_size(current_database()) / (1024.0 * 1024.0)"))

	assert.Greater(t, skippedSizeMb, skippedTableMinSizeMb)
	assert.Greater(t, backup.RawDatabaseSizeMb, 0.0)
	assert.LessOrEqual(t, backup.RawDatabaseSizeMb+skippedSizeMb, wholeDatabaseSizeMb, message)
}

// postgresqlCycle describes one backup of a new database and its restore into a new database of the
// same server.
type postgresqlCycle struct {
	settings     apiclient.PostgresqlLogicalSettings
	encryption   apiclient.BackupEncryption
	changeTarget func(target *apiclient.PostgresqlRestoreTarget)
	// prepareRestoredDatabase runs against the empty target before the restore, nil for none.
	prepareRestoredDatabase func(t *testing.T, restoredDB *sqlx.DB)
}

type restoredPostgresqlCycle struct {
	tm         enginetest.Team
	backup     apiclient.LogicalBackup
	restoredDB *sqlx.DB
}

func (s postgresqlServer) backUpAndRestore(t *testing.T, cycle postgresqlCycle) restoredPostgresqlCycle {
	t.Helper()

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, cycle.settings)
	backup := tm.BackUp(t, database.Id, cycle.encryption)

	restoredDatabaseName, restoredDB := s.createRestoreDatabase(t, "restored")
	if cycle.prepareRestoredDatabase != nil {
		cycle.prepareRestoredDatabase(t, restoredDB)
	}

	target := s.newTarget(restoredDatabaseName)
	if cycle.changeTarget != nil {
		cycle.changeTarget(&target)
	}

	tm.Restore(t, backup.Id, toPostgresqlRestoreTarget(t, target))

	return restoredPostgresqlCycle{tm: tm, backup: backup, restoredDB: restoredDB}
}

func toPostgresqlRestoreTarget(t *testing.T, target apiclient.PostgresqlRestoreTarget) apiclient.RestoreTarget {
	t.Helper()

	var restoreTarget apiclient.RestoreTarget
	require.NoError(t, restoreTarget.FromPostgresqlRestoreTarget(target))

	return restoreTarget
}

func withPostgresqlCredentials(
	t *testing.T,
	database apiclient.Database,
	username string,
	password string,
) apiclient.PostgresqlLogicalSettings {
	t.Helper()

	settings, err := database.Settings.AsPostgresqlLogicalSettings()
	require.NoError(t, err)

	settings.Username = username
	settings.Password = &password

	return settings
}

func createAndFillPostgresqlTableQuery(tableName string) string {
	return fmt.Sprintf(`
DROP TABLE IF EXISTS %[1]s;
CREATE TABLE %[1]s (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    value INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO %[1]s (name, value) VALUES ('test1', 100), ('test2', 200), ('test3', 300);
`, tableName)
}

func verifyPostgresqlRows(t *testing.T, sourceDB, restoredDB *sqlx.DB, tableName string) {
	t.Helper()

	var sourceRows []postgresqlTestRow
	require.NoError(
		t,
		sourceDB.SelectContext(t.Context(), &sourceRows, fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName)),
	)

	var restoredRows []postgresqlTestRow
	require.NoError(t, restoredDB.SelectContext(t.Context(), &restoredRows,
		fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName)))

	require.Len(t, restoredRows, len(sourceRows), "the restored table has as many rows as the source")

	for rowIndex := range sourceRows {
		assert.Equal(t, sourceRows[rowIndex].ID, restoredRows[rowIndex].ID, "id")
		assert.Equal(t, sourceRows[rowIndex].Name, restoredRows[rowIndex].Name, "name")
		assert.Equal(t, sourceRows[rowIndex].Value, restoredRows[rowIndex].Value, "value")
	}
}

func isPostgresqlTableRestored(t *testing.T, db *sqlx.DB, relation relationRef) bool {
	t.Helper()

	var isTablePresent bool
	require.NoError(t, db.GetContext(t.Context(), &isTablePresent, `
		SELECT EXISTS (
			SELECT FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2 AND c.relkind IN ('r', 'p')
		)
	`, relation.SchemaName, relation.TableName))

	return isTablePresent
}

func isPostgresqlSchemaRestored(t *testing.T, db *sqlx.DB, schemaName string) bool {
	t.Helper()

	var isSchemaPresent bool
	require.NoError(t, db.GetContext(t.Context(), &isSchemaPresent,
		"SELECT EXISTS (SELECT FROM pg_namespace WHERE nspname = $1)", schemaName))

	return isSchemaPresent
}

func isPostgresqlExtensionInstalled(t *testing.T, db *sqlx.DB, extensionName string) bool {
	t.Helper()

	var isExtensionPresent bool
	require.NoError(t, db.GetContext(t.Context(), &isExtensionPresent,
		"SELECT EXISTS (SELECT FROM pg_extension WHERE extname = $1)", extensionName))

	return isExtensionPresent
}

func generatePostgresqlUUID(t *testing.T, db *sqlx.DB) string {
	t.Helper()

	var generatedUUID string
	require.NoError(t, db.GetContext(t.Context(), &generatedUUID, "SELECT uuid_generate_v4()::text"))

	return generatedUUID
}

func countPostgresqlRows(t *testing.T, db *sqlx.DB, relation relationRef) int {
	t.Helper()

	var rowCount int
	require.NoError(t, db.GetContext(t.Context(), &rowCount,
		fmt.Sprintf(`SELECT count(*) FROM %q.%q`, relation.SchemaName, relation.TableName)))

	return rowCount
}

func getPostgresqlIndexNames(t *testing.T, db *sqlx.DB, relation relationRef) []string {
	t.Helper()

	var indexNames []string
	require.NoError(t, db.SelectContext(t.Context(), &indexNames, `
		SELECT indexname FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY indexname
	`, relation.SchemaName, relation.TableName))

	return indexNames
}

func getPostgresqlColumnDefinitions(t *testing.T, db *sqlx.DB, relation relationRef) []string {
	t.Helper()

	var columnDefinitions []string
	require.NoError(t, db.SelectContext(t.Context(), &columnDefinitions, `
		SELECT column_name || ' ' || data_type || ' ' || is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, relation.SchemaName, relation.TableName))

	return columnDefinitions
}

func hasPostgresqlPrimaryKey(t *testing.T, db *sqlx.DB, relation relationRef) bool {
	t.Helper()

	var isPrimaryKeyPresent bool
	require.NoError(t, db.GetContext(t.Context(), &isPrimaryKeyPresent, `
		SELECT EXISTS (
			SELECT FROM pg_constraint
			WHERE contype = 'p'
			  AND conrelid = format('%I.%I', $1::text, $2::text)::regclass
		)
	`, relation.SchemaName, relation.TableName))

	return isPrimaryKeyPresent
}

func readMtlsCertificates(t *testing.T) (clientCert, clientKey, rootCert string) {
	t.Helper()

	read := func(fileName string) string {
		content, err := os.ReadFile(filepath.Join(enginetest.GetTestdataDirectory(t, "mtls"), fileName))
		require.NoError(t, err)

		return string(content)
	}

	return read("client.crt"), read("client.key"), read("ca.crt")
}

// newMtlsConnectionOptions writes the PEMs to files of mode 0600, because lib/pq refuses a client key
// that others can read, and returns the libpq options that verify the server and present the client.
func newMtlsConnectionOptions(t *testing.T, clientCert, clientKey, rootCert string) string {
	t.Helper()

	certificateDirectory := t.TempDir()
	for fileName, content := range map[string]string{
		"client.crt": clientCert,
		"client.key": clientKey,
		"ca.crt":     rootCert,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(certificateDirectory, fileName), []byte(content), 0o600))
	}

	return fmt.Sprintf("sslmode=verify-ca sslrootcert=%s sslcert=%s sslkey=%s",
		filepath.ToSlash(filepath.Join(certificateDirectory, "ca.crt")),
		filepath.ToSlash(filepath.Join(certificateDirectory, "client.crt")),
		filepath.ToSlash(filepath.Join(certificateDirectory, "client.key")),
	)
}

func getNonNilList(list []string) []string {
	if list == nil {
		return []string{}
	}

	return list
}
