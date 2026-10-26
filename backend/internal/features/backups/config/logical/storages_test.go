package backups_config_logical

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/period"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_AttachStorageFromSameWorkspace_SuccessfullyAttached(t *testing.T) {
	router := createTestRouterWithStorage()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	timeOfDay := "04:00"
	request := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: &BackupConfigStorage{ID: storage.ID},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	var response LogicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.NotNil(t, response.StorageID)
	assert.Equal(t, storage.ID, *response.StorageID)
}

func Test_AttachStorageFromDifferentWorkspace_ReturnsForbidden(t *testing.T) {
	router := createTestRouterWithStorage()

	owner1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1)
	database := createTestDatabaseViaAPI("Test Database", workspace1.ID, owner1.Token, router)

	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2)
	storage := createTestStorage(workspace2.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2)
	}()

	timeOfDay := "04:00"
	request := LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		Storage: &BackupConfigStorage{ID: storage.ID},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/save",
		"Bearer "+owner1.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage does not belong to the same workspace")
}

func createTestRouterWithStorage() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases.GetDatabaseController(),
		GetBackupConfigController(),
	)

	databases.SetupDependencies()
	SetupDependencies()

	return router
}
