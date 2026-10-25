package restores

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	env_config "databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/features/notifiers"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/restoring"
	"databasus-backend/internal/features/storages"
	local_storage "databasus-backend/internal/features/storages/models/local"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_dto "databasus-backend/internal/features/users/dto"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/cache"
	util_encryption "databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

func Test_GetRestores_WhenUserIsWorkspaceMember_RestoresReturned(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	var restores []*restores_core.Restore
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&restores,
	)

	assert.NotNil(t, restores)
	assert.Equal(t, 0, len(restores))
	assert.NotNil(t, database)
}

func Test_GetRestores_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	testResp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+nonMember.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_GetRestores_WhenUserIsGlobalAdmin_RestoresReturned(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	var restores []*restores_core.Restore
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s", backup.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
		&restores,
	)

	assert.NotNil(t, restores)
}

func Test_RestoreBackup_WhenUserIsWorkspaceMember_RestoreInitiated(t *testing.T) {
	router := createTestRouter()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "restore started successfully")
}

func Test_RestoreBackup_WhenUserIsNotWorkspaceMember_ReturnsForbidden(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+nonMember.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "insufficient permissions")
}

func Test_RestoreBackup_WithIsExcludeExtensions_FlagPassedCorrectly(t *testing.T) {
	router := createTestRouter()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:             tools.PostgresqlVersion16,
			Host:                env_config.GetEnv().TestLocalhost,
			Port:                5432,
			Username:            "postgres",
			Password:            "postgres",
			IsExcludeExtensions: true,
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "restore started successfully")
}

func Test_RestoreBackup_AuditLogWritten(t *testing.T) {
	router := createTestRouter()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	time.Sleep(100 * time.Millisecond)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{
			Limit:  100,
			Offset: 0,
		},
	)
	assert.NoError(t, err)

	found := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Database restored for database") &&
			strings.Contains(log.Message, database.Name) {
			found = true
			break
		}
	}
	assert.True(t, found, "Audit log for restore not found")
}

func Test_RestoreBackup_DiskSpaceValidation(t *testing.T) {
	tests := []struct {
		name                string
		dbType              databases.DatabaseType
		cpuCount            int
		expectDiskValidated bool
	}{
		{
			name:                "PostgreSQL_CPU4_SpaceValidated",
			dbType:              databases.DatabaseTypePostgresLogical,
			cpuCount:            4,
			expectDiskValidated: true,
		},
		{
			name:                "PostgreSQL_CPU1_SpaceNotValidated",
			dbType:              databases.DatabaseTypePostgresLogical,
			cpuCount:            1,
			expectDiskValidated: false,
		},
		{
			name:                "MySQL_SpaceNotValidated",
			dbType:              databases.DatabaseTypeMysql,
			cpuCount:            3,
			expectDiskValidated: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := createTestRouter()

			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
			defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

			var database *databases.Database
			var backup *backups_core_logical.LogicalBackup
			var storage *storages.Storage
			var request restores_core.RestoreBackupRequest

			if tc.dbType == databases.DatabaseTypePostgresLogical {
				database, backup = createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
				defer cleanupDatabaseWithBackup(t.Context(), database, backup)
				request = restores_core.RestoreBackupRequest{
					PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
						Version:  tools.PostgresqlVersion16,
						Host:     env_config.GetEnv().TestLocalhost,
						Port:     5432,
						Username: "postgres",
						Password: "postgres",
						CpuCount: tc.cpuCount,
					},
				}
			} else {
				mysqlDB := createTestMySQLDatabase(
					t,
					"Test MySQL DB",
					workspace.ID,
					owner.Token,
					router,
				)
				database = mysqlDB
				storage = createTestStorage(workspace.ID)

				defer func() {
					// Cleanup in dependency order: backup -> database -> storage
					cleanupBackup(backup)
					databases.RemoveTestDatabase(t.Context(), mysqlDB)
					time.Sleep(50 * time.Millisecond)
					storages.RemoveTestStorage(t.Context(), storage.ID)
				}()

				configService := backups_config_logical.GetBackupConfigService()
				config, err := configService.GetBackupConfigByDbId(mysqlDB.ID)
				assert.NoError(t, err)

				config.IsBackupsEnabled = true
				config.StorageID = &storage.ID
				config.Storage = storage
				_, err = configService.SaveBackupConfig(t.Context(), config)
				assert.NoError(t, err)

				backup = createTestBackup(mysqlDB, storage)

				request = restores_core.RestoreBackupRequest{
					MysqlDatabase: &mysql.MysqlDatabase{
						Version:  tools.MysqlVersion80,
						Host:     env_config.GetEnv().TestLocalhost,
						Port:     3306,
						Username: "root",
						Password: "password",
					},
				}
			}

			// Set huge backup size (10 TB) that would fail disk validation if checked
			repo := &backups_core_logical.BackupRepository{}
			backup.BackupSizeMb = 10485760.0
			err := repo.Save(backup)
			assert.NoError(t, err)

			expectedStatus := http.StatusOK
			if tc.expectDiskValidated {
				expectedStatus = http.StatusBadRequest
			}

			testResp := test_utils.MakePostRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
				"Bearer "+owner.Token,
				request,
				expectedStatus,
			)

			bodyStr := string(testResp.Body)
			if tc.expectDiskValidated {
				assert.Contains(t, bodyStr, "is required")
				assert.Contains(t, bodyStr, "is available")
				assert.Contains(t, bodyStr, "disk space")
			} else {
				assert.Contains(t, bodyStr, "restore started successfully")
			}
		})
	}
}

func Test_CancelRestore_InProgressRestore_SuccessfullyCancelled(t *testing.T) {
	cache.GetStore().Clear(t.Context())
	tasks_cancellation.SetupDependencies()

	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	router := createTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backupRepo := backups_core_logical.BackupRepository{}
		backups, _ := backupRepo.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepo.DeleteByID(backup.ID)
		}

		restoreRepo := restores_core.RestoreRepository{}
		restores, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusInProgress)
		for _, restore := range restores {
			restoreRepo.DeleteByID(restore.ID)
		}
		restores, _ = restoreRepo.FindByStatus(restores_core.RestoreStatusCanceled)
		for _, restore := range restores {
			restoreRepo.DeleteByID(restore.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

		cache.GetStore().Clear(t.Context())
	}()

	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage)
	backup := backups_controllers_logical.CreateTestBackup(database.ID, storage.ID)

	mockUsecase := &restoring.MockBlockingRestoreUsecase{
		StartedChan: make(chan bool, 1),
	}
	restorer := restoring.CreateTestRestorerWithUsecase(mockUsecase)

	cancelRestorer := restoring.StartRestorerForTest(t, restorer)
	defer cancelRestorer()

	time.Sleep(200 * time.Millisecond)

	restoreRequest := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	var restoreResponse map[string]any
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+user.Token,
		restoreRequest,
		http.StatusOK,
		&restoreResponse,
	)

	select {
	case <-mockUsecase.StartedChan:
		t.Log("Restore started and is blocking")
	case <-time.After(2 * time.Second):
		t.Fatal("Restore did not start within timeout")
	}

	restoreRepo := &restores_core.RestoreRepository{}
	restores, err := restoreRepo.FindByBackupID(backup.ID)
	assert.NoError(t, err)
	assert.Greater(t, len(restores), 0, "At least one restore should exist")

	var restoreID uuid.UUID
	for _, r := range restores {
		if r.Status == restores_core.RestoreStatusInProgress {
			restoreID = r.ID
			break
		}
	}
	assert.NotEqual(t, uuid.Nil, restoreID, "Should find an in-progress restore")

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/cancel/%s", restoreID.String()),
		"Bearer "+user.Token,
		nil,
		http.StatusNoContent,
	)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	deadline := time.Now().UTC().Add(3 * time.Second)
	var restore *restores_core.Restore
	for time.Now().UTC().Before(deadline) {
		restore, err = restoreRepo.FindByID(restoreID)
		assert.NoError(t, err)
		if restore.Status == restores_core.RestoreStatusCanceled {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, restores_core.RestoreStatusCanceled, restore.Status)

	auditLogService := audit_logs.GetAuditLogService()
	auditLogs, err := auditLogService.GetWorkspaceAuditLogs(t.Context(),
		workspace.ID,
		&audit_logs.GetAuditLogsRequest{Limit: 100, Offset: 0},
	)
	assert.NoError(t, err)

	foundCancelLog := false
	for _, log := range auditLogs.AuditLogs {
		if strings.Contains(log.Message, "Restore cancelled") &&
			strings.Contains(log.Message, database.Name) {
			foundCancelLog = true
			break
		}
	}
	assert.True(t, foundCancelLog, "Cancel audit log should be created")

	time.Sleep(200 * time.Millisecond)
}

func Test_RestoreBackup_WithParallelRestoreInProgress_ReturnsError(t *testing.T) {
	router := createTestRouter()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	// Seeded rather than started for real: a restore dispatched over HTTP runs in a goroutine that
	// reaches a terminal status within milliseconds, so the IN_PROGRESS state the guard reacts to
	// would be a race.
	restoreInProgress := restoring.CreateTestRestore(t, backup, restores_core.RestoreStatusInProgress)
	defer func() {
		restoreRepository := &restores_core.RestoreRepository{}
		if err := restoreRepository.DeleteByID(restoreInProgress.ID); err != nil {
			t.Logf("Failed to delete seeded restore: %v", err)
		}
	}()

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},
	}

	response := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "another restore is already in progress")
}

func createTestRouter() *gin.Engine {
	return CreateTestRouter()
}

func createTestDatabaseWithBackupForRestore(
	ctx context.Context,
	workspace *workspaces_models.Workspace,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*databases.Database, *backups_core_logical.LogicalBackup) {
	database := createTestDatabase("Test Database", workspace.ID, owner.Token, router)
	storage := createTestStorage(workspace.ID)

	configService := backups_config_logical.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	if err != nil {
		panic(err)
	}

	config.IsBackupsEnabled = true
	config.StorageID = &storage.ID
	config.Storage = storage
	_, err = configService.SaveBackupConfig(ctx, config)
	if err != nil {
		panic(err)
	}

	backup := createTestBackup(database, storage)

	return database, backup
}

func createTestDatabase(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	request := databases.Database{
		WorkspaceID:       &workspaceID,
		Name:              name,
		Type:              databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: databases.GetTestPostgresConfig(),
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		panic(
			fmt.Sprintf("Failed to create database. Status: %d, Body: %s", w.Code, w.Body.String()),
		)
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func createTestMySQLDatabase(
	t *testing.T,
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	endpoint := containers.StartMysql(t, "mysql:8.0")

	testDbName := "testdb"
	request := databases.Database{
		WorkspaceID: &workspaceID,
		Name:        name,
		Type:        databases.DatabaseTypeMysql,
		Mysql: &mysql.MysqlDatabase{
			Version:  tools.MysqlVersion80,
			Host:     endpoint.Host,
			Port:     endpoint.Port,
			Username: containers.MysqlUsername,
			Password: containers.MysqlPassword,
			Database: &testDbName,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		panic(
			fmt.Sprintf(
				"Failed to create MySQL database. Status: %d, Body: %s",
				w.Code,
				w.Body.String(),
			),
		)
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func createTestStorage(workspaceID uuid.UUID) *storages.Storage {
	storage := &storages.Storage{
		WorkspaceID:  workspaceID,
		Type:         storages.StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}

	repo := &storages.StorageRepository{}
	storage, err := repo.Save(storage)
	if err != nil {
		panic(err)
	}

	return storage
}

func createTestBackup(
	database *databases.Database,
	storage *storages.Storage,
) *backups_core_logical.LogicalBackup {
	fieldEncryptor := util_encryption.GetFieldEncryptor()

	backup := &backups_core_logical.LogicalBackup{
		ID:               uuid.New(),
		DatabaseID:       database.ID,
		StorageID:        storage.ID,
		Status:           backups_core_logical.BackupStatusCompleted,
		BackupSizeMb:     10.5,
		BackupDurationMs: 1000,
		CreatedAt:        time.Now().UTC(),
	}

	repo := &backups_core_logical.BackupRepository{}
	if err := repo.Save(backup); err != nil {
		panic(err)
	}

	dummyContent := []byte("dummy backup content for testing")
	reader := strings.NewReader(string(dummyContent))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := storage.SaveFile(
		context.Background(),
		fieldEncryptor,
		logger,
		backup.ID.String(),
		reader,
	); err != nil {
		panic(fmt.Sprintf("Failed to create test backup file: %v", err))
	}

	return backup
}

func cleanupDatabaseWithBackup(
	ctx context.Context,
	database *databases.Database,
	backup *backups_core_logical.LogicalBackup,
) {
	// Clean up in reverse dependency order
	cleanupBackup(backup)
	databases.RemoveTestDatabase(ctx, database)
	time.Sleep(50 * time.Millisecond)

	// Clean up storage last (after database and backup are removed)
	configService := backups_config_logical.GetBackupConfigService()
	config, err := configService.GetBackupConfigByDbId(database.ID)
	if err == nil && config.StorageID != nil {
		storages.RemoveTestStorage(ctx, *config.StorageID)
	}
}

func Test_RestoreBackup_WithCpuCountMoreThanOne_RestoreInitiated(t *testing.T) {
	router := createTestRouter()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	database, backup := createTestDatabaseWithBackupForRestore(t.Context(), workspace, owner, router)
	defer cleanupDatabaseWithBackup(t.Context(), database, backup)

	request := restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     env_config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
			CpuCount: 4,
		},
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backup.ID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)

	assert.Contains(t, string(testResp.Body), "restore started successfully")
}

func cleanupBackup(backup *backups_core_logical.LogicalBackup) {
	repo := &backups_core_logical.BackupRepository{}
	repo.DeleteByID(backup.ID)
}
