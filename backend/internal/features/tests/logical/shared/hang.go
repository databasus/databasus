//go:build testhelpers

package logicaltesting

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
)

// SetupFlakyBackupTarget builds the common fixture for the issue-582 regression:
// a router, a member user, a workspace, and a deliberately-flaky S3 storage whose
// SaveFile fails. s3Endpoint is a reachable MinIO URL ("http://host:port") with a
// missing bucket. It registers cleanup for the storage and workspace and returns
// the router, the user's token, and the workspace and storage IDs. The caller
// then attaches a database and runs AssertBackupFailsWithoutHanging.
func SetupFlakyBackupTarget(
	t *testing.T,
	workspaceName string,
	s3Endpoint string,
) (router *gin.Engine, token string, workspaceID, storageID uuid.UUID) {
	t.Helper()

	router = CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), workspaceName, user, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(context.Background(), workspace, router) })

	// The bucket deliberately does not exist on this (reachable) MinIO, so SaveFile fails at upload
	// rather than at connect — that is the issue-582 path the assertion below guards.
	storage := storages.CreateTestFlakyS3Storage(workspace.ID, s3Endpoint)
	t.Cleanup(func() { storages.RemoveTestStorage(t.Context(), storage.ID) })

	return router, user.Token, workspace.ID, storage.ID
}

// AssertBackupFailsWithoutHanging enables backups to a (flaky) storage, triggers
// a backup, and asserts it reaches Failed with a fail message within a bounded
// time instead of hanging — the regression for issue #582. The database is
// deleted on return. Each engine package calls it from its own hang test with an
// already-created database whose backups are expected to fail at SaveFile.
func AssertBackupFailsWithoutHanging(
	t *testing.T,
	router *gin.Engine,
	token string,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) {
	t.Helper()

	defer test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+databaseID.String(),
		"Bearer "+token,
		http.StatusNoContent,
	)

	EnableBackupsViaAPI(
		t, router, databaseID, storageID,
		backups_core_enums.BackupEncryptionNone, token,
	)

	CreateBackupViaAPI(t, router, databaseID, token)

	backup := WaitForBackupTerminalStatus(t, router, databaseID, token, 2*time.Minute)

	require.Equalf(
		t,
		backups_core_logical.BackupStatusFailed,
		backup.Status,
		"issue #582: backup should be marked Failed when SaveFile fails; got status=%s",
		backup.Status,
	)
	require.NotNil(
		t,
		backup.FailMessage,
		"issue #582: failed backup must carry a fail message describing the SaveFile error",
	)
}
