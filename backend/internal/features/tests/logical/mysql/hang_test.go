package mysql_logical

import (
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	mysql_engine "databasus-backend/internal/engines/mysql"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	"databasus-backend/internal/util/testing/containers"
)

// Test_BackupShouldFailNotHang_Mysql_RegressionForIssue582 asserts a MySQL backup
// to a flaky S3 storage fails (with a message) instead of hanging.
func Test_BackupShouldFailNotHang_Mysql_RegressionForIssue582(t *testing.T) {
	minioEndpoint := containers.StartMinio(t)
	router, token, workspaceID, storageID := logicaltesting.SetupFlakyBackupTarget(
		t, "Issue 582 MySQL Workspace", fmt.Sprintf("http://%s:%d", minioEndpoint.Host, minioEndpoint.Port),
	)

	container, err := connectToMysqlContainer(t, "mysql:8.0", mysql_engine.Version80)
	if err != nil {
		t.Fatalf("failed to connect to MySQL test container: %v", err)
	}
	defer container.DB.Close()

	setupMysqlTestData(t, container.DB)

	database := createMysqlDatabaseViaAPI(
		t, router, "Issue 582 MySQL DB", workspaceID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.Version,
		token,
	)

	logicaltesting.AssertBackupFailsWithoutHanging(t, router, token, database.Id, storageID)
}
