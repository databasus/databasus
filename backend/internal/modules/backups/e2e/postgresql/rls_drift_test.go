package postgresql

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func testPostgresqlBackupFailsAfterRLSDrift(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	roleName := "rls_reader_" + enginetest.NewUniqueSuffix()
	password := uuid.NewString()
	server.cleanUp(t, `DROP ROLE IF EXISTS `+roleName)
	databaseName, source := server.createRestoreDatabase(t, "rls_drift")
	_, err := source.ExecContext(t.Context(), fmt.Sprintf(`
		CREATE ROLE %s LOGIN PASSWORD '%s';
		CREATE TABLE a_readable (id INT, payload TEXT);
		INSERT INTO a_readable SELECT n, repeat('ordinary row ', 100) FROM generate_series(1, 2000) n;
		CREATE TABLE z_tenant_rows (id INT, tenant_id INT);
		INSERT INTO z_tenant_rows VALUES (1, 1), (2, 2), (3, 2);
		GRANT USAGE ON SCHEMA public TO %s;
		GRANT SELECT ON ALL TABLES IN SCHEMA public TO %s;
	`, roleName, password, roleName, roleName))
	require.NoError(t, err)

	settings := server.newSettings()
	settings.Database = &databaseName
	settings.Username = roleName
	settings.Password = &password
	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, settings)

	baselineBackup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
	tm.App.WaitForBackupRuns(t)
	baselineFileName := getRLSBackupFileName(t, tm, baselineBackup)
	require.True(t, tm.App.IsLocalStorageFilePresent(t, baselineFileName))
	require.True(t, tm.App.IsLocalStorageFilePresent(t, baselineFileName+".metadata"))

	_, err = source.ExecContext(t.Context(), `ALTER TABLE z_tenant_rows ENABLE ROW LEVEL SECURITY;
		CREATE POLICY tenant_access ON z_tenant_rows USING (tenant_id = 1)`)
	require.NoError(t, err)

	restrictedSource, err := sqlx.ConnectContext(t.Context(), "postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		endpoint.Host,
		endpoint.Port,
		roleName,
		password,
		databaseName,
	))
	require.NoError(t, err)
	defer func() { _ = restrictedSource.Close() }()

	var visibleRows, totalRows int
	require.NoError(t, restrictedSource.GetContext(t.Context(), &visibleRows, `SELECT count(*) FROM z_tenant_rows`))
	require.NoError(t, source.GetContext(t.Context(), &totalRows, `SELECT count(*) FROM z_tenant_rows`))
	require.Equal(t, 1, visibleRows)
	require.Equal(t, 3, totalRows)

	failedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, database.Id).Id)

	require.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status)
	require.NotNil(t, failedBackup.FailMessage)
	assert.Contains(t, *failedBackup.FailMessage, "row-level security")
	assert.Contains(t, *failedBackup.FailMessage, "z_tenant_rows")
	assert.NotContains(t, *failedBackup.FailMessage, password)

	tm.App.WaitForBackupRuns(t)
	failedFileName := getRLSBackupFileName(t, tm, failedBackup)
	tm.App.RunJobPass(t, "storage_file_deletion")

	assert.False(t, tm.App.IsLocalStorageFilePresent(t, failedFileName))
	assert.False(t, tm.App.IsLocalStorageFilePresent(t, failedFileName+".metadata"))
	assert.False(t, tm.App.IsLocalStorageStagingFilePresent(t, failedFileName))
	assert.True(t, tm.App.IsLocalStorageFilePresent(t, baselineFileName))
	assert.True(t, tm.App.IsLocalStorageFilePresent(t, baselineFileName+".metadata"))

	baselineResponse, err := tm.Owner.Client.GetLogicalBackupWithResponse(t.Context(), baselineBackup.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, baselineResponse.StatusCode())
	assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, baselineResponse.JSON200.Status)
	tm.RequireBackupSizeMatchesDownload(t, baselineBackup)
}

func getRLSBackupFileName(t *testing.T, tm enginetest.Team, backup apiclient.LogicalBackup) string {
	t.Helper()

	var fileName string
	tm.App.QuerySQL(t, &fileName, `SELECT file_name FROM backups_logical WHERE id = ?`, backup.Id)
	require.NotEmpty(t, fileName)

	return fileName
}
