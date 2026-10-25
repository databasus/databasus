package postgresql_logical

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

type backupScopeSettings struct {
	IncludeSchemas   []string
	ExcludeTables    []string
	ExcludeTableData []string
}

type restoredBackupScope struct {
	Backup     *backups_core_logical.LogicalBackup
	RestoredDB *sqlx.DB
}

func backupAndRestoreWithScope(
	t *testing.T,
	container *PostgresContainer,
	pgVersion string,
	settings backupScopeSettings,
) restoredBackupScope {
	t.Helper()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Backup Scope Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	database := createDatabaseViaAPI(
		t, router, "Backup Scope Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		user.Token,
	)

	t.Cleanup(func() {
		test_utils.MakeDeleteRequest(
			t, router, "/api/v1/databases/"+database.ID.String(),
			"Bearer "+user.Token, http.StatusNoContent,
		)
		storages.RemoveTestStorage(context.Background(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router)
	})

	database.PostgresqlLogical.IncludeSchemas = settings.IncludeSchemas
	database.PostgresqlLogical.ExcludeTables = settings.ExcludeTables
	database.PostgresqlLogical.ExcludeTableData = settings.ExcludeTableData
	updateDatabaseViaAPI(t, router, database, user.Token)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	require.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	})

	restoredDBName := fmt.Sprintf("restored_scope_%s_%s", pgVersion, uuid.New().String()[:8])
	_, err := container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", restoredDBName))
	require.NoError(t, err)

	restoredDB, err := sqlx.Connect("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, restoredDBName,
	))
	require.NoError(t, err)

	t.Cleanup(func() {
		restoredDB.Close()
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", restoredDBName))
	})

	createRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password, restoredDBName,
		user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	require.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	return restoredBackupScope{Backup: backup, RestoredDB: restoredDB}
}

func seedBackupScopeStatements(t *testing.T, container *PostgresContainer, statements string) {
	t.Helper()

	_, err := container.DB.Exec(statements)
	require.NoError(t, err)
}

func isTableRestored(t *testing.T, db *sqlx.DB, relation relationRef) bool {
	t.Helper()

	var isTablePresent bool
	err := db.Get(&isTablePresent, `
		SELECT EXISTS (
			SELECT FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND c.relname = $2 AND c.relkind IN ('r', 'p')
		)
	`, relation.SchemaName, relation.TableName)
	require.NoError(t, err)

	return isTablePresent
}

func isSchemaRestored(t *testing.T, db *sqlx.DB, schemaName string) bool {
	t.Helper()

	var isSchemaPresent bool
	err := db.Get(
		&isSchemaPresent,
		`SELECT EXISTS (SELECT FROM pg_namespace WHERE nspname = $1)`,
		schemaName,
	)
	require.NoError(t, err)

	return isSchemaPresent
}

func countTableRows(t *testing.T, db *sqlx.DB, relation relationRef) int {
	t.Helper()

	var rowCount int
	err := db.Get(
		&rowCount,
		fmt.Sprintf(`SELECT count(*) FROM %q.%q`, relation.SchemaName, relation.TableName),
	)
	require.NoError(t, err)

	return rowCount
}

func getTableIndexNames(t *testing.T, db *sqlx.DB, relation relationRef) []string {
	t.Helper()

	var indexNames []string
	err := db.Select(&indexNames, `
		SELECT indexname FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY indexname
	`, relation.SchemaName, relation.TableName)
	require.NoError(t, err)

	return indexNames
}

func getTableColumnDefinitions(t *testing.T, db *sqlx.DB, relation relationRef) []string {
	t.Helper()

	var columnDefinitions []string
	err := db.Select(&columnDefinitions, `
		SELECT column_name || ' ' || data_type || ' ' || is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, relation.SchemaName, relation.TableName)
	require.NoError(t, err)

	return columnDefinitions
}

func hasPrimaryKey(t *testing.T, db *sqlx.DB, relation relationRef) bool {
	t.Helper()

	var isPrimaryKeyPresent bool
	err := db.Get(&isPrimaryKeyPresent, `
		SELECT EXISTS (
			SELECT FROM pg_constraint
			WHERE contype = 'p'
			  AND conrelid = format('%I.%I', $1::text, $2::text)::regclass
		)
	`, relation.SchemaName, relation.TableName)
	require.NoError(t, err)

	return isPrimaryKeyPresent
}

func testBackupRestoreWithExcludeTableDataForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	require.NoError(t, err)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	emptiedTable := relationRef{SchemaName: "public", TableName: "logs_" + suffix}
	neighbourTable := relationRef{SchemaName: "public", TableName: "orders_" + suffix}

	seedBackupScopeStatements(t, container, fmt.Sprintf(`
		CREATE TABLE public.%[1]s (id SERIAL PRIMARY KEY, payload TEXT NOT NULL);
		CREATE INDEX %[1]s_payload_idx ON public.%[1]s (payload);
		INSERT INTO public.%[1]s (payload) SELECT repeat('x', 1024) FROM generate_series(1, 20000);
		CREATE TABLE public.%[2]s (id SERIAL PRIMARY KEY, data TEXT NOT NULL);
		INSERT INTO public.%[2]s (data) VALUES ('first'), ('second');
	`, emptiedTable.TableName, neighbourTable.TableName))
	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(
			"DROP TABLE IF EXISTS public.%s, public.%s",
			emptiedTable.TableName, neighbourTable.TableName,
		))
	})

	sourceIndexNames := getTableIndexNames(t, container.DB, emptiedTable)
	require.Len(t, sourceIndexNames, 2)
	sourceColumnDefinitions := getTableColumnDefinitions(t, container.DB, emptiedTable)

	restored := backupAndRestoreWithScope(t, container, pgVersion, backupScopeSettings{
		ExcludeTableData: []string{"public." + emptiedTable.TableName},
	})

	assert.True(t, isTableRestored(t, restored.RestoredDB, emptiedTable))
	assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, emptiedTable))
	assert.Equal(t, sourceIndexNames, getTableIndexNames(t, restored.RestoredDB, emptiedTable))
	assert.Equal(
		t, sourceColumnDefinitions, getTableColumnDefinitions(t, restored.RestoredDB, emptiedTable),
	)
	assert.True(t, hasPrimaryKey(t, restored.RestoredDB, emptiedTable))
	assert.Equal(t, 2, countTableRows(t, restored.RestoredDB, neighbourTable))

	emptiedTableSizeMb := measureTableSizeMb(t, container.DB, emptiedTable)
	assert.Greater(t, emptiedTableSizeMb, skippedFromDumpTableMinSizeMb)
	assert.Greater(t, restored.Backup.BackupRawDbSizeMb, 0.0)
	assert.LessOrEqual(
		t,
		restored.Backup.BackupRawDbSizeMb+emptiedTableSizeMb,
		measureWholeDatabaseSizeMb(t, container.DB),
		"recorded raw db size must not count the table whose data is excluded",
	)
}

func testBackupRestoreWithExclusionPatternsForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	require.NoError(t, err)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	archiveSchema := "archive_" + suffix
	historyPrefix := "history_" + suffix
	logsTable := "logs_" + suffix
	auditTable := "audit_tmp_" + suffix
	overlapTable := "overlap_" + suffix
	oldOrdersTable := "old_orders_" + suffix

	seedBackupScopeStatements(t, container, fmt.Sprintf(`
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
	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(
			"DROP SCHEMA IF EXISTS %s CASCADE; "+
				"DROP TABLE IF EXISTS public.%s_2024, public.%s, public.%s, public.%s, public.%s",
			archiveSchema, historyPrefix, logsTable, auditTable, overlapTable, oldOrdersTable,
		))
	})

	restored := backupAndRestoreWithScope(t, container, pgVersion, backupScopeSettings{
		ExcludeTables: []string{
			auditTable,
			"public." + overlapTable,
			"public.old_orders_" + suffix[:4] + "*",
		},
		ExcludeTableData: []string{
			"*." + historyPrefix + "_*",
			logsTable,
			"public." + overlapTable,
		},
	})

	publicHistoryTable := relationRef{SchemaName: "public", TableName: historyPrefix + "_2024"}
	archiveHistoryTable := relationRef{SchemaName: archiveSchema, TableName: historyPrefix + "_2025"}
	assert.True(t, isTableRestored(t, restored.RestoredDB, publicHistoryTable))
	assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, publicHistoryTable))
	assert.True(t, isTableRestored(t, restored.RestoredDB, archiveHistoryTable))
	assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, archiveHistoryTable))

	publicLogsTable := relationRef{SchemaName: "public", TableName: logsTable}
	archiveLogsTable := relationRef{SchemaName: archiveSchema, TableName: logsTable}
	assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, publicLogsTable))
	assert.Equal(
		t, 2, countTableRows(t, restored.RestoredDB, archiveLogsTable),
		"an unqualified data exclusion must not reach a table off the default search path",
	)

	assert.False(t, isTableRestored(
		t, restored.RestoredDB, relationRef{SchemaName: "public", TableName: auditTable},
	))
	assert.Equal(
		t, 2,
		countTableRows(t, restored.RestoredDB, relationRef{SchemaName: archiveSchema, TableName: auditTable}),
		"an unqualified whole-table exclusion must not reach a table off the default search path",
	)

	assert.False(t, isTableRestored(
		t, restored.RestoredDB, relationRef{SchemaName: "public", TableName: oldOrdersTable},
	))
	assert.Equal(
		t, 2,
		countTableRows(t, restored.RestoredDB, relationRef{SchemaName: archiveSchema, TableName: oldOrdersTable}),
		"a schema-qualified whole-table pattern must not reach another schema",
	)

	assert.False(
		t,
		isTableRestored(t, restored.RestoredDB, relationRef{SchemaName: "public", TableName: overlapTable}),
		"a table in both lists must be excluded entirely",
	)
}

func testBackupRestoreWithExcludeTableDataOnPartitionsForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	require.NoError(t, err)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	parentNamedAlone := "events_a_" + suffix
	partitionsByPattern := "events_b_" + suffix

	partitionedTableStatements := func(parentName string) string {
		return fmt.Sprintf(`
			CREATE TABLE public.%[1]s (id INT NOT NULL) PARTITION BY RANGE (id);
			CREATE TABLE public.%[1]s_2025 PARTITION OF public.%[1]s FOR VALUES FROM (0) TO (100);
			CREATE TABLE public.%[1]s_2026 PARTITION OF public.%[1]s FOR VALUES FROM (100) TO (200);
			INSERT INTO public.%[1]s VALUES (1), (2), (101);
		`, parentName)
	}

	seedBackupScopeStatements(
		t, container,
		partitionedTableStatements(parentNamedAlone)+partitionedTableStatements(partitionsByPattern),
	)
	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(
			"DROP TABLE IF EXISTS public.%s, public.%s", parentNamedAlone, partitionsByPattern,
		))
	})

	restored := backupAndRestoreWithScope(t, container, pgVersion, backupScopeSettings{
		ExcludeTableData: []string{
			"public." + parentNamedAlone,
			"public." + partitionsByPattern + "_*",
		},
	})

	assert.Equal(
		t,
		2,
		countTableRows(
			t,
			restored.RestoredDB,
			relationRef{SchemaName: "public", TableName: parentNamedAlone + "_2025"},
		),
		"naming only the partitioned parent must keep its partitions' rows",
	)
	assert.Equal(
		t,
		1,
		countTableRows(
			t,
			restored.RestoredDB,
			relationRef{SchemaName: "public", TableName: parentNamedAlone + "_2026"},
		),
	)

	for _, tableName := range []string{
		partitionsByPattern,
		partitionsByPattern + "_2025",
		partitionsByPattern + "_2026",
	} {
		relation := relationRef{SchemaName: "public", TableName: tableName}

		assert.True(t, isTableRestored(t, restored.RestoredDB, relation), tableName)
		assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, relation), tableName)
	}
}

func testBackupRestoreWithSchemasAndExcludeTableDataForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	pgVersion string,
) {
	container, err := connectToPostgresEndpoint(t, endpoint)
	require.NoError(t, err)
	t.Cleanup(func() { container.DB.Close() })

	suffix := uuid.New().String()[:8]
	reportingSchema := "reporting_" + suffix
	logsTable := "logs_" + suffix

	seedBackupScopeStatements(t, container, fmt.Sprintf(`
		CREATE SCHEMA %[1]s;
		CREATE TABLE public.%[2]s (id INT);
		CREATE TABLE %[1]s.%[2]s (id INT);
		INSERT INTO public.%[2]s VALUES (1);
		INSERT INTO %[1]s.%[2]s VALUES (1);
	`, reportingSchema, logsTable))
	t.Cleanup(func() {
		_, _ = container.DB.Exec(fmt.Sprintf(
			"DROP SCHEMA IF EXISTS %s CASCADE; DROP TABLE IF EXISTS public.%s",
			reportingSchema, logsTable,
		))
	})

	restored := backupAndRestoreWithScope(t, container, pgVersion, backupScopeSettings{
		IncludeSchemas:   []string{"public"},
		ExcludeTableData: []string{logsTable},
	})

	publicLogsTable := relationRef{SchemaName: "public", TableName: logsTable}
	assert.True(t, isTableRestored(t, restored.RestoredDB, publicLogsTable))
	assert.Equal(t, 0, countTableRows(t, restored.RestoredDB, publicLogsTable))
	assert.False(t, isSchemaRestored(t, restored.RestoredDB, reportingSchema))
}
