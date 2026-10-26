package mariadb_logical

import (
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	"databasus-backend/internal/util/testing/containers"
)

// Test_BackupShouldFailNotHang_Mariadb_RegressionForIssue582 asserts a MariaDB
// backup to a flaky S3 storage fails (with a message) instead of hanging.
func Test_BackupShouldFailNotHang_Mariadb_RegressionForIssue582(t *testing.T) {
	minioEndpoint := containers.StartMinio(t)
	router, token, workspaceID, storageID := logicaltesting.SetupFlakyBackupTarget(
		t, "Issue 582 MariaDB Workspace", fmt.Sprintf("http://%s:%d", minioEndpoint.Host, minioEndpoint.Port),
	)

	container, err := connectToMariadbContainer(t, "mariadb:10.11", mariadb_engine.Version1011)
	if err != nil {
		t.Fatalf("failed to connect to MariaDB test container: %v", err)
	}
	defer container.DB.Close()

	setupMariadbTestData(t, container.DB)

	database := createMariadbDatabaseViaAPI(
		t, router, "Issue 582 MariaDB DB", workspaceID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		token,
	)

	logicaltesting.AssertBackupFailsWithoutHanging(t, router, token, database.Id, storageID)
}
