package mongodb_logical

import (
	"fmt"
	"testing"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	"databasus-backend/internal/util/testing/containers"
)

// Test_BackupShouldFailNotHang_Mongodb_RegressionForIssue582 asserts a MongoDB
// backup to a flaky S3 storage fails (with a message) instead of hanging.
func Test_BackupShouldFailNotHang_Mongodb_RegressionForIssue582(t *testing.T) {
	minioEndpoint := containers.StartMinio(t)
	router, token, workspaceID, storageID := logicaltesting.SetupFlakyBackupTarget(
		t, "Issue 582 MongoDB Workspace", fmt.Sprintf("http://%s:%d", minioEndpoint.Host, minioEndpoint.Port),
	)

	container := connectToMongodbContainer(t, "mongo:7.0", mongodb_engine.Version7)
	defer func() { _ = container.Client.Disconnect(t.Context()) }()

	setupMongodbTestData(t, container)

	database := createMongodbDatabaseViaAPI(
		t, router, "Issue 582 MongoDB DB", workspaceID,
		container.Host, container.Port,
		container.Username, container.Password, container.Database,
		container.AuthDatabase,
		container.Version,
		token,
	)

	logicaltesting.AssertBackupFailsWithoutHanging(t, router, token, database.ID, storageID)
}
