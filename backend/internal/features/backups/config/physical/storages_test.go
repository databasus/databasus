package backups_config_physical

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_AttachStorageFromSameWorkspace_PhysicalConfig_SuccessfullyAttached(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createPhysicalDatabaseViaAPI(
		t, "Physical DB "+uuid.New().String(), workspace.ID, owner.Token, router,
		postgresql_physical.BackupTypeFullOnly, "17",
	)
	storage := storages.CreateTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	request.Storage = storage

	var response PhysicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.NotNil(t, response.StorageID)
	assert.Equal(t, storage.ID, *response.StorageID)
}

func Test_AttachStorageFromDifferentWorkspace_PhysicalConfig_ReturnsBadRequest(t *testing.T) {
	router := createPhysicalTestRouter()

	owner1 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1, router)
	database := createPhysicalDatabaseViaAPI(
		t, "Physical DB "+uuid.New().String(), workspace1.ID, owner1.Token, router,
		postgresql_physical.BackupTypeFullOnly, "17",
	)

	owner2 := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2, router)
	storage := storages.CreateTestStorage(workspace2.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	request.Storage = storage

	testResp := test_utils.MakePostRequest(
		t, router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner1.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage does not belong to the same workspace")
}

func Test_DeleteStorageWithAttachedPhysicalConfig_CannotDelete(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	database := createPhysicalDatabaseViaAPI(
		t, "Physical DB "+uuid.New().String(), workspace.ID, owner.Token, router,
		postgresql_physical.BackupTypeFullOnly, "17",
	)
	storage := storages.CreateTestStorage(workspace.ID)

	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	request.Storage = storage

	var response PhysicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t, router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	testResp := test_utils.MakeDeleteRequest(
		t, router,
		fmt.Sprintf("/api/v1/storages/%s", storage.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "storage has attached databases and cannot be deleted")
}
