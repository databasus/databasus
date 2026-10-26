package postgresql_logical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	"databasus-backend/internal/util/testing/containers"
)

// Test_BackupShouldFailNotHang_Postgresql_RegressionForIssue582 asserts a
// PostgreSQL backup to a flaky S3 storage fails (with a message) instead of
// hanging.
func Test_BackupShouldFailNotHang_Postgresql_RegressionForIssue582(t *testing.T) {
	minioEndpoint := containers.StartMinio(t)
	router, token, workspaceID, storageID := logicaltesting.SetupFlakyBackupTarget(
		t, "Issue 582 PostgreSQL Workspace", fmt.Sprintf("http://%s:%d", minioEndpoint.Host, minioEndpoint.Port),
	)

	container, err := connectToPostgresContainer(t, "postgres:16")
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL test container: %v", err)
	}
	defer container.DB.Close()

	_, err = container.DB.Exec(createAndFillTableQuery("test_data"))
	require.NoError(t, err)

	database := createDatabaseViaAPI(
		t, router, "Issue 582 PostgreSQL DB", workspaceID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		token,
	)

	logicaltesting.AssertBackupFailsWithoutHanging(t, router, token, database.Id, storageID)
}
