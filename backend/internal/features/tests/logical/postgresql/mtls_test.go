package postgresql_logical

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

func Test_BackupAndRestorePostgresqlMtls_Succeeds(t *testing.T) {
	contextDir, err := filepath.Abs(filepath.Join("testdata", "mtls"))
	if err != nil {
		t.Fatalf("failed to resolve mtls testdata dir: %v", err)
	}

	endpoint := containers.StartPostgresMtls(t, contextDir)
	host := endpoint.Host
	portInt := endpoint.Port

	clientCert, clientKey, rootCert := readMtlsCerts(t)
	certDir := writeMtlsCertFiles(t, clientCert, clientKey, rootCert)

	originalDB, err := sqlx.Connect("postgres", buildMtlsDSN(host, portInt, "testdb", certDir))
	if err != nil {
		t.Fatalf("failed to connect to mTLS Postgres: %v", err)
	}
	defer originalDB.Close()

	tableName := "test_data_mtls"
	_, err = originalDB.Exec(createAndFillTableQuery(tableName))
	assert.NoError(t, err)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Postgres mTLS Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)

	dbName := "testdb"
	database := createPostgresqlMtlsDatabaseViaAPI(
		t, router, "Postgres mTLS DB", workspace.ID,
		host, portInt, "testuser", "testpassword", dbName,
		clientCert, clientKey, rootCert, user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.ID, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)
	logicaltesting.CreateBackupViaAPI(t, router, database.ID, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.ID, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_pg_mtls"
	_, err = originalDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)
	_, err = originalDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	newDB, err := sqlx.Connect("postgres", buildMtlsDSN(host, portInt, newDBName, certDir))
	assert.NoError(t, err)
	defer newDB.Close()

	createPostgresqlMtlsRestoreViaAPI(
		t, router, backup.ID,
		host, portInt, "testuser", "testpassword", newDBName,
		clientCert, clientKey, rootCert, user.Token,
	)

	restore := waitForRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyDataIntegrity(t, originalDB, newDB, tableName)

	_ = os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String()))
	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/databases/"+database.ID.String(),
		"Bearer "+user.Token, http.StatusNoContent,
	)
	storages.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_CreatePostgresqlMtls_WhenClientCertMissing_IsRejected(t *testing.T) {
	contextDir, err := filepath.Abs(filepath.Join("testdata", "mtls"))
	if err != nil {
		t.Fatalf("failed to resolve mtls testdata dir: %v", err)
	}

	endpoint := containers.StartPostgresMtls(t, contextDir)
	host := endpoint.Host
	portInt := endpoint.Port

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Postgres mTLS Reject Workspace", user, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	dbName := "testdb"
	request := databases.Database{
		Name:        "Postgres mTLS no client cert",
		WorkspaceID: &workspace.ID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{
			Host:     host,
			Port:     portInt,
			Username: "testuser",
			Password: "testpassword",
			Database: &dbName,
			SslMode:  postgresql_shared.PostgresSslModeRequire,
			CpuCount: 1,
		},
	}

	response := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+user.Token, request,
	)
	assert.NotEqual(
		t, http.StatusCreated, response.Code,
		"creating an mTLS database without a client certificate must fail",
	)
}

func readMtlsCerts(t *testing.T) (clientCert, clientKey, rootCert string) {
	t.Helper()

	read := func(name string) string {
		content, err := os.ReadFile(filepath.Join("testdata", "mtls", name))
		if err != nil {
			t.Fatalf("failed to read mTLS test certificate %s: %v", name, err)
		}

		return string(content)
	}

	return read("client.crt"), read("client.key"), read("ca.crt")
}

// writeMtlsCertFiles writes the PEMs to a 0600 temp directory because the lib/pq
// driver refuses a client key file that is group- or world-readable.
func writeMtlsCertFiles(t *testing.T, clientCert, clientKey, rootCert string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range map[string]string{
		"client.crt": clientCert,
		"client.key": clientKey,
		"ca.crt":     rootCert,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return dir
}

func buildMtlsDSN(host string, port int, dbName, certDir string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=%s "+
			"sslmode=verify-ca sslrootcert=%s sslcert=%s sslkey=%s",
		host, port, dbName,
		filepath.ToSlash(filepath.Join(certDir, "ca.crt")),
		filepath.ToSlash(filepath.Join(certDir, "client.crt")),
		filepath.ToSlash(filepath.Join(certDir, "client.key")),
	)
}

func createPostgresqlMtlsDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username, password, database string,
	clientCert, clientKey, rootCert, token string,
) *databases.Database {
	request := databases.Database{
		Name:        name,
		WorkspaceID: &workspaceID,
		Type:        databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{
			Host:          host,
			Port:          port,
			Username:      username,
			Password:      password,
			Database:      &database,
			SslMode:       postgresql_shared.PostgresSslModeVerifyCA,
			SslClientCert: clientCert,
			SslClientKey:  clientKey,
			SslRootCert:   rootCert,
			CpuCount:      1,
		},
	}

	return logicaltesting.SubmitCreateDatabase(t, router, "Postgres mTLS", request, token)
}

func createPostgresqlMtlsRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username, password, database string,
	clientCert, clientKey, rootCert, token string,
) {
	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &pgtypes.PostgresqlLogicalDatabase{
			Host:          host,
			Port:          port,
			Username:      username,
			Password:      password,
			Database:      &database,
			SslMode:       postgresql_shared.PostgresSslModeVerifyCA,
			SslClientCert: clientCert,
			SslClientKey:  clientKey,
			SslRootCert:   rootCert,
			CpuCount:      1,
		},
	}
	logicaltesting.SubmitRestore(t, router, backupID, request, token)
}
