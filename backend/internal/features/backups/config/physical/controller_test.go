package backups_config_physical

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
)

func createPhysicalTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		databases.GetDatabaseController(),
		GetBackupConfigController(),
	)

	databases.SetupDependencies()
	SetupDependencies()

	return router
}

func createPhysicalDatabaseViaAPI(
	t *testing.T,
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
	backupType postgresql_physical.BackupType,
	versionTag string,
) *databases.Database {
	t.Helper()

	env := config.GetEnv()

	var portStr string
	var version pgshared.Version

	switch versionTag {
	case "17":
		portStr = env.TestPhysicalPostgres17Port
		version = pgshared.Version17
	case "18":
		portStr = env.TestPhysicalPostgres18Port
		version = pgshared.Version18
	default:
		t.Fatalf("unsupported physical postgres version tag: %s", versionTag)
	}

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	request := databases.Database{
		WorkspaceID: &workspaceID,
		Name:        name,
		Type:        databases.DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    version,
			Host:       env.TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: backupType,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)
	require.Equal(t, http.StatusCreated, w.Code, "create physical database failed: %s", w.Body.String())

	var database databases.Database
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))

	return &database
}

func validPhysicalConfigForFullOnly(databaseID uuid.UUID) PhysicalBackupConfig {
	return PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: true,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: new("04:00"),
		},
		Retention: RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  7,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
	}
}

func Test_SaveBackupConfig_PhysicalWithDifferentRoles_EnforcesPermissions(t *testing.T) {
	tests := []struct {
		name               string
		workspaceRole      *workspaces.Role
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "workspace owner can save physical backup config",
			workspaceRole:      new(workspaces.RoleOwner),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace admin can save physical backup config",
			workspaceRole:      new(workspaces.RoleAdmin),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace member can save physical backup config",
			workspaceRole:      new(workspaces.RoleMember),
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "workspace viewer cannot save physical backup config",
			workspaceRole:      new(workspaces.RoleViewer),
			expectSuccess:      false,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "global admin can save physical backup config",
			workspaceRole:      nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createPhysicalTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

			database := createPhysicalDatabaseViaAPI(
				t,
				"Physical DB "+uuid.New().String(),
				workspace.ID,
				owner.Token,
				router,
				postgresql_physical.BackupTypeFullOnly,
				"17",
			)

			defer func() {
				databases.RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
			}()

			var testUserToken string

			switch {
			case tt.isGlobalAdmin:
				admin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
				testUserToken = admin.Token
			case tt.workspaceRole != nil && *tt.workspaceRole == workspaces.RoleOwner:
				testUserToken = owner.Token
			case tt.workspaceRole != nil:
				member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace, member, *tt.workspaceRole)

				testUserToken = member.Token
			}

			request := validPhysicalConfigForFullOnly(database.ID)

			var response PhysicalBackupConfig
			testResp := test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/backup-configs/physical/save",
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
				&response,
			)

			if tt.expectSuccess {
				assert.Equal(t, database.ID, response.DatabaseID)
				assert.True(t, response.IsBackupsEnabled)
				assert.Equal(t, RetentionFullBackups, response.Retention)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
			}
		})
	}
}

func Test_SaveBackupConfig_PhysicalFullOnlyWithChainsRetention_ReturnsBadRequest(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	request.Retention = RetentionChains
	request.ChainsRetention = ChainsRetention{Count: 3}
	request.FullBackupsRetention = FullBackupsRetention{}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "FULL_ONLY")
}

func Test_SaveBackupConfig_PhysicalFullOnlyWithIncrementalInterval_ReturnsBadRequest(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	request := validPhysicalConfigForFullOnly(database.ID)
	incrementalTime := "02:00"
	request.IncrementalBackupInterval = intervals.Interval{
		Type:      intervals.IntervalHourly,
		TimeOfDay: &incrementalTime,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "incremental cadence cannot be set")
}

func Test_GetBackupConfig_PhysicalWhenNoneExists_InitializesDefaults(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	var response PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.False(t, response.IsBackupsEnabled)
	assert.Equal(t, RetentionFullBackups, response.Retention)
	assert.Equal(t, FullBackupsRetentionPolicyLastN, response.FullBackupsRetention.Policy)
	assert.Equal(t, 7, response.FullBackupsRetention.Count)
	assert.Contains(t, response.SendNotificationsOn, NotificationChainBroken)
}

type recordingBackupCancellationListener struct {
	backupsDisabledCount       int
	walStreamingDisabledCount  int
	backupsStorageChangedCount int
}

func (r *recordingBackupCancellationListener) OnBackupsStorageChanged(_ context.Context, _ uuid.UUID) {
	r.backupsStorageChangedCount++
}

func (r *recordingBackupCancellationListener) OnBackupsDisabled(_ context.Context, _ uuid.UUID) {
	r.backupsDisabledCount++
}

func (r *recordingBackupCancellationListener) OnWalStreamingDisabled(_ context.Context, _ uuid.UUID) {
	r.walStreamingDisabledCount++
}

func recordBackupCancellations(t *testing.T) *recordingBackupCancellationListener {
	t.Helper()

	previousListener := GetBackupConfigService().backupCancellationListener

	recorder := &recordingBackupCancellationListener{}
	GetBackupConfigService().SetBackupCancellationListener(recorder)
	t.Cleanup(func() { GetBackupConfigService().SetBackupCancellationListener(previousListener) })

	return recorder
}

func validPhysicalConfigForWalStream(databaseID uuid.UUID) PhysicalBackupConfig {
	return PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: true,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalWeekly,
			Weekday:   new(1),
			TimeOfDay: new("04:00"),
		},
		IncrementalBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: new("05:00"),
		},
		Retention:            RetentionChains,
		ChainsRetention:      ChainsRetention{Count: 3},
		WalLagThresholdBytes: 64 * 1024 * 1024,
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationWalGap,
		},
	}
}

type backupTypeUpdateSpec struct {
	router     *gin.Engine
	token      string
	database   *databases.Database
	backupType postgresql_physical.BackupType
}

func updateBackupTypeViaAPI(t *testing.T, spec backupTypeUpdateSpec) {
	t.Helper()

	physicalSettings := *spec.database.PostgresqlPhysical
	physicalSettings.BackupType = spec.backupType

	updateRequest := *spec.database
	updateRequest.PostgresqlPhysical = &physicalSettings

	test_utils.MakePostRequest(
		t,
		spec.router,
		"/api/v1/databases/update",
		"Bearer "+spec.token,
		updateRequest,
		http.StatusOK,
	)
}

// https://github.com/databasus/databasus/issues/746
func Test_SaveBackupConfig_AfterBackupTypeDemotedFromWalStream_ConfigSavesCleanly(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	recorder := recordBackupCancellations(t)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		validPhysicalConfigForWalStream(database.ID),
		http.StatusOK,
	)

	updateBackupTypeViaAPI(t, backupTypeUpdateSpec{
		router:     router,
		token:      owner.Token,
		database:   database,
		backupType: postgresql_physical.BackupTypeFullAndIncremental,
	})

	var demotedConfig PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&demotedConfig,
	)

	assert.Zero(t, demotedConfig.WalLagThresholdBytes)
	assert.Equal(t, RetentionChains, demotedConfig.Retention)
	assert.Equal(t, 1, recorder.walStreamingDisabledCount, "demotion must stand the wal streamer down")

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		demotedConfig,
		http.StatusOK,
	)
}

func Test_SaveBackupConfig_AfterBackupTypePromotedFromFullOnly_ConfigSavesCleanly(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		validPhysicalConfigForFullOnly(database.ID),
		http.StatusOK,
	)

	updateBackupTypeViaAPI(t, backupTypeUpdateSpec{
		router:     router,
		token:      owner.Token,
		database:   database,
		backupType: postgresql_physical.BackupTypeFullIncrementalAndWalStream,
	})

	var promotedConfig PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&promotedConfig,
	)

	assert.Equal(t, RetentionChainsAndFullBackups, promotedConfig.Retention)
	assert.Equal(t, defaultChainsRetentionCount, promotedConfig.ChainsRetention.Count)
	assert.Equal(t, int64(defaultWalLagThresholdBytes), promotedConfig.WalLagThresholdBytes)
	assert.Equal(t, intervals.IntervalHourly, promotedConfig.IncrementalBackupInterval.Type)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		promotedConfig,
		http.StatusOK,
	)
}

func Test_GetBackupConfig_PhysicalIncrementalWhenNoneExists_InitializesDefaultsThatSave(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullAndIncremental,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	var initializedConfig PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&initializedConfig,
	)

	assert.Equal(t, RetentionChainsAndFullBackups, initializedConfig.Retention)
	assert.Equal(t, defaultChainsRetentionCount, initializedConfig.ChainsRetention.Count)
	assert.Zero(t, initializedConfig.WalLagThresholdBytes)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		initializedConfig,
		http.StatusOK,
	)
}

func Test_SaveBackupConfig_WhenBackupsGetDisabled_StandsBackupWorkDown(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	recorder := recordBackupCancellations(t)

	enabledConfig := validPhysicalConfigForFullOnly(database.ID)
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		enabledConfig,
		http.StatusOK,
	)
	assert.Equal(t, 0, recorder.backupsDisabledCount)

	disabledConfig := enabledConfig
	disabledConfig.IsBackupsEnabled = false

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		disabledConfig,
		http.StatusOK,
	)

	assert.Equal(t, 1, recorder.backupsDisabledCount)
	assert.Equal(t, 0, recorder.walStreamingDisabledCount)
}

// https://github.com/databasus/databasus/issues/746, as left on installations
// that switched backup type before the config followed along.
func Test_GetBackupConfig_WhenStoredConfigPredatesTheBackupType_RepairsItOnRead(t *testing.T) {
	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)

	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		"17",
	)
	defer func() {
		databases.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	}()

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		validPhysicalConfigForWalStream(database.ID),
		http.StatusOK,
	)

	require.NoError(t, storage.GetDb().
		Model(&postgresql_physical.PostgresqlPhysicalDatabase{}).
		Where("database_id = ?", database.ID).
		Update("backup_type", postgresql_physical.BackupTypeFullAndIncremental).Error)

	var repairedConfig PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/backup-configs/physical/database/%s", database.ID),
		"Bearer "+owner.Token,
		http.StatusOK,
		&repairedConfig,
	)

	assert.Zero(t, repairedConfig.WalLagThresholdBytes)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/backup-configs/physical/save",
		"Bearer "+owner.Token,
		repairedConfig,
		http.StatusOK,
	)
}

type physicalStorageReferenceFixture struct {
	router    *gin.Engine
	ownerAuth string
	workspace uuid.UUID
	database  *databases.Database
	// Removed after the database, whose backup config refers to them.
	createdStorageIDs []uuid.UUID
}

func (f *physicalStorageReferenceFixture) createS3Storage() storages.Storage {
	createdStorage := storages_bridge.CreateTestFlakyS3Storage(f.workspace, "http://127.0.0.1:1")
	f.createdStorageIDs = append(f.createdStorageIDs, createdStorage.ID)

	return createdStorage
}

func createPhysicalStorageReferenceFixture(t *testing.T) *physicalStorageReferenceFixture {
	t.Helper()

	router := createPhysicalTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	database := createPhysicalDatabaseViaAPI(
		t,
		"Physical DB "+uuid.New().String(),
		workspace.ID,
		owner.Token,
		router,
		postgresql_physical.BackupTypeFullOnly,
		"17",
	)

	fixture := &physicalStorageReferenceFixture{
		router:    router,
		ownerAuth: "Bearer " + owner.Token,
		workspace: workspace.ID,
		database:  database,
	}

	t.Cleanup(func() {
		databases.RemoveTestDatabase(t.Context(), database)
		for _, storageID := range fixture.createdStorageIDs {
			storages_bridge.RemoveTestStorage(t.Context(), storageID)
		}
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	})

	return fixture
}

func Test_SaveBackupConfig_PhysicalWhenEmbeddedStorageHasChangedFields_KeepsStoredStorage(t *testing.T) {
	fixture := createPhysicalStorageReferenceFixture(t)
	existingStorage := fixture.createS3Storage()
	storedKeys := getStoredS3Keys(t, existingStorage.ID)

	var savedConfig PhysicalBackupConfig
	test_utils.MakePostRequestAndUnmarshal(
		t,
		fixture.router,
		"/api/v1/backup-configs/physical/save",
		fixture.ownerAuth,
		newPhysicalConfigWithStorage(t, fixture.database.ID, map[string]any{
			"id":   existingStorage.ID,
			"name": "Renamed through the backup config",
			"type": storages.TypeS3,
			"s3Storage": map[string]any{
				"s3AccessKey": "replaced-access-key",
				"s3SecretKey": "replaced-secret-key",
			},
		}),
		http.StatusOK,
		&savedConfig,
	)

	require.NotNil(t, savedConfig.StorageID)
	assert.Equal(t, existingStorage.ID, *savedConfig.StorageID)

	storedStorages, err := storages_bridge.GetModule().GetStorages(t.Context(), []uuid.UUID{existingStorage.ID})
	require.NoError(t, err)
	require.Len(t, storedStorages, 1)
	assert.Equal(t, existingStorage.Name, storedStorages[0].Name)
	assert.Equal(t, storedKeys, getStoredS3Keys(t, existingStorage.ID))
}

func Test_SaveBackupConfig_PhysicalWhenEmbeddedStorageHasNoID_ReturnsBadRequestAndCreatesNoStorage(
	t *testing.T,
) {
	fixture := createPhysicalStorageReferenceFixture(t)

	resp := test_utils.MakePostRequest(
		t,
		fixture.router,
		"/api/v1/backup-configs/physical/save",
		fixture.ownerAuth,
		newPhysicalConfigWithStorage(t, fixture.database.ID, map[string]any{
			"workspaceId":  fixture.workspace,
			"type":         storages.TypeLocal,
			"name":         "Storage created through the backup config",
			"localStorage": map[string]any{},
		}),
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), ErrStorageWithoutID.Error())
	assert.Zero(t, countStoragesInWorkspace(t, fixture.workspace))
}

func Test_SaveBackupConfig_PhysicalWhenEmbeddedStorageIsUnknown_ReturnsBadRequestAndCreatesNoStorage(
	t *testing.T,
) {
	fixture := createPhysicalStorageReferenceFixture(t)

	test_utils.MakePostRequest(
		t,
		fixture.router,
		"/api/v1/backup-configs/physical/save",
		fixture.ownerAuth,
		newPhysicalConfigWithStorage(t, fixture.database.ID, map[string]any{
			"id":           uuid.New(),
			"workspaceId":  fixture.workspace,
			"type":         storages.TypeLocal,
			"name":         "Unknown storage",
			"localStorage": map[string]any{},
		}),
		http.StatusBadRequest,
	)

	assert.Zero(t, countStoragesInWorkspace(t, fixture.workspace))
}

func Test_SaveBackupConfig_PhysicalWhenStorageChanges_StopsStreamingOnlyOnChange(t *testing.T) {
	fixture := createPhysicalStorageReferenceFixture(t)
	firstStorage := fixture.createS3Storage()
	secondStorage := fixture.createS3Storage()
	recorder := recordBackupCancellations(t)

	saveConfigWithStorage := func(embeddedStorage storages.Storage) {
		backupConfig := validPhysicalConfigForFullOnly(fixture.database.ID)
		backupConfig.Storage = &BackupConfigStorage{ID: embeddedStorage.ID}

		test_utils.MakePostRequest(
			t,
			fixture.router,
			"/api/v1/backup-configs/physical/save",
			fixture.ownerAuth,
			backupConfig,
			http.StatusOK,
		)
	}

	saveConfigWithStorage(firstStorage)
	streamingStopsAfterFirstStorage := recorder.backupsStorageChangedCount

	saveConfigWithStorage(firstStorage)
	assert.Equal(t, streamingStopsAfterFirstStorage, recorder.backupsStorageChangedCount,
		"saving with the same storage leaves streaming alone")

	saveConfigWithStorage(secondStorage)
	assert.Equal(t, streamingStopsAfterFirstStorage+1, recorder.backupsStorageChangedCount,
		"moving to another storage stops the streamer on the old one")
}

// newPhysicalConfigWithStorage embeds the storage as the frontend sent it before configs carried only
// a reference, so a test can show that extra storage fields change nothing.
func newPhysicalConfigWithStorage(t *testing.T, databaseID uuid.UUID, embeddedStorage map[string]any) map[string]any {
	t.Helper()

	encodedConfig, err := json.Marshal(validPhysicalConfigForFullOnly(databaseID))
	require.NoError(t, err)

	var configWithStorage map[string]any
	require.NoError(t, json.Unmarshal(encodedConfig, &configWithStorage))
	configWithStorage["storage"] = embeddedStorage

	return configWithStorage
}

func countStoragesInWorkspace(t *testing.T, workspaceID uuid.UUID) int64 {
	t.Helper()

	var storagesInWorkspace int64
	require.NoError(t, storage.GetDb().
		Raw("SELECT COUNT(*) FROM storages WHERE workspace_id = ?", workspaceID).
		Scan(&storagesInWorkspace).Error)

	return storagesInWorkspace
}

// getStoredS3Keys reads the encrypted keys, which no operation returns.
func getStoredS3Keys(t *testing.T, storageID uuid.UUID) []string {
	t.Helper()

	var storedKeys []string
	require.NoError(t, storage.GetDb().
		Raw("SELECT s3_access_key || ' ' || s3_secret_key FROM storages_s3 WHERE storage_id = ?", storageID).
		Scan(&storedKeys).Error)
	require.Len(t, storedKeys, 1)

	return storedKeys
}
