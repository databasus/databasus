package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/email"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	local_storage "databasus-backend/internal/features/storages/models/local"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	users_controllers "databasus-backend/internal/features/users/controllers"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	test_utils "databasus-backend/internal/util/testing"
)

const (
	seedPassword  = "Q1234567"
	otherPassword = "Other-password-1"

	firstBackupTimeout = 2 * time.Minute
)

func Test_Seed_WhenAccountIsNew_CreatesWorkspaceStorageAndDatabasesWithBackups(t *testing.T) {
	router, params := setUpSeedTest(t)

	exitCode, output := runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	token := signInForTest(t, router, params.Email, seedPassword)
	workspaceID := findWorkspaceID(t, router, token, params.WorkspaceName)
	require.NotEqual(t, uuid.Nil, workspaceID, "workspace %q was not created", params.WorkspaceName)

	storageID := findStorageID(t, router, token, workspaceID, params.StorageName)
	require.NotEqual(t, uuid.Nil, storageID, "storage %q was not created", params.StorageName)

	logicalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName)
	walDatabaseID := findDatabaseID(t, router, token, workspaceID, params.WalDatabaseName)
	require.NotEqual(t, uuid.Nil, logicalDatabaseID)
	require.NotEqual(t, uuid.Nil, walDatabaseID)

	var logicalConfig backups_config_logical.LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backup-configs/database/"+logicalDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &logicalConfig)

	assert.True(t, logicalConfig.IsBackupsEnabled)
	assert.Equal(t, storageID, *logicalConfig.StorageID)
	assert.Equal(t, backups_config_logical.RetentionPolicyTypeCount, logicalConfig.RetentionPolicyType)
	assert.Equal(t, 3, logicalConfig.RetentionCount)

	var logicalBackups backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backups?database_id="+logicalDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &logicalBackups)
	assert.NotEmpty(t, logicalBackups.Backups, "the first logical backup was not started")

	var walConfig backups_config_physical.PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backup-configs/physical/database/"+walDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &walConfig)

	assert.True(t, walConfig.IsBackupsEnabled)
	assert.Equal(t, storageID, *walConfig.StorageID)
	assert.Equal(t, backups_config_physical.RetentionChains, walConfig.Retention)
	assert.Equal(t, 2, walConfig.ChainsRetention.Count)

	storedWalConfig, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(walDatabaseID)
	require.NoError(t, err)
	assert.NotNil(t, storedWalConfig.ForceFullRequestedAt, "the first WAL-streaming backup was not requested")
}

func Test_Seed_WhenRunTwice_CreatesNothingAndKeepsEditedDatabase(t *testing.T) {
	router, params := setUpSeedTest(t)

	exitCode, output := runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	token := signInForTest(t, router, params.Email, seedPassword)
	workspaceID := findWorkspaceID(t, router, token, params.WorkspaceName)
	logicalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName)
	waitForLogicalBackups(t, logicalDatabaseID)

	var editedConfig backups_config_logical.LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backup-configs/database/"+logicalDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &editedConfig)

	editedConfig.RetentionCount = 9
	editedConfig.IsBackupsEnabled = false
	test_utils.MakePostRequest(t, router, "/api/v1/backup-configs/save",
		"Bearer "+token, editedConfig, http.StatusOK)

	itemsBefore := listSeededItems(t, router, token)

	exitCode, output = runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	assert.Equal(t, itemsBefore, listSeededItems(t, router, token))

	var configAfter backups_config_logical.LogicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backup-configs/database/"+logicalDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &configAfter)

	assert.Equal(t, 9, configAfter.RetentionCount)
	assert.False(t, configAfter.IsBackupsEnabled)
}

func Test_Seed_WhenSeededDatabaseWasDeleted_RecreatesOnlyThatDatabase(t *testing.T) {
	router, params := setUpSeedTest(t)

	exitCode, output := runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	token := signInForTest(t, router, params.Email, seedPassword)
	workspaceID := findWorkspaceID(t, router, token, params.WorkspaceName)
	storageID := findStorageID(t, router, token, workspaceID, params.StorageName)
	logicalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName)
	deletedWalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.WalDatabaseName)

	test_utils.MakeDeleteRequest(t, router, "/api/v1/databases/"+deletedWalDatabaseID.String(),
		"Bearer "+token, http.StatusNoContent)

	exitCode, output = runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	recreatedWalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.WalDatabaseName)
	require.NotEqual(t, uuid.Nil, recreatedWalDatabaseID)
	assert.NotEqual(t, deletedWalDatabaseID, recreatedWalDatabaseID)

	var walConfig backups_config_physical.PhysicalBackupConfig
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/backup-configs/physical/database/"+recreatedWalDatabaseID.String(),
		"Bearer "+token, http.StatusOK, &walConfig)
	assert.True(t, walConfig.IsBackupsEnabled)

	assert.Equal(t, workspaceID, findWorkspaceID(t, router, token, params.WorkspaceName))
	assert.Equal(t, storageID, findStorageID(t, router, token, workspaceID, params.StorageName))
	assert.Equal(t, logicalDatabaseID, findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName))
	assert.Len(t, listSeededItems(t, router, token).databaseIDs, 2)
}

func Test_Seed_WhenAccountHasAnotherPassword_FailsAndLeavesAccountUnchanged(t *testing.T) {
	router, params := setUpSeedTest(t)

	signUpForTest(t, router, params.Email, otherPassword)
	token := signInForTest(t, router, params.Email, otherPassword)
	profileBefore := getProfile(t, router, token)

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.Email)
	assert.Contains(t, output, "password is incorrect")

	tokenAfter := signInForTest(t, router, params.Email, otherPassword)
	assert.Equal(t, profileBefore, getProfile(t, router, tokenAfter))
	assert.Empty(t, listSeededItems(t, router, tokenAfter).workspaceIDs)
}

func Test_Seed_WhenAnotherAccountExists_LeavesThatAccountUnchanged(t *testing.T) {
	router, params := setUpSeedTest(t)

	otherEmail := "seed-other-" + uuid.NewString() + "@example.com"
	signUpForTest(t, router, otherEmail, otherPassword)
	t.Cleanup(func() { removeAccountWithItems(t, router, otherEmail) })

	otherProfileBefore := getProfile(t, router, signInForTest(t, router, otherEmail, otherPassword))

	exitCode, output := runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	otherToken := signInForTest(t, router, otherEmail, otherPassword)
	assert.Equal(t, otherProfileBefore, getProfile(t, router, otherToken))
	assert.Empty(t, listSeededItems(t, router, otherToken).workspaceIDs)
}

func Test_Seed_WhenTwoFactorSignInIsRequired_FailsAndCreatesNothing(t *testing.T) {
	router, params := setUpSeedTest(t)

	token := signUpForTest(t, router, params.Email, seedPassword)

	users_services.GetUserService().SetEmailSender(users_testing.NewMockEmailSender())
	users_testing.EnableTwoFactorAuth(t.Context())
	t.Cleanup(func() {
		users_services.GetUserService().SetEmailSender(email.GetEmailSMTPSender())
		users_testing.ResetSettingsToDefaults(context.Background())
	})

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.Email)
	assert.Contains(t, output, "two-factor sign-in")
	assert.Empty(t, listSeededItems(t, router, token).workspaceIDs)
}

func Test_Seed_WhenMemberMayNotCreateWorkspaces_FailsNamingWorkspace(t *testing.T) {
	router, params := setUpSeedTest(t)

	token := signUpForTest(t, router, params.Email, seedPassword)
	require.Equal(t, users_enums.UserRoleMember, getProfile(t, router, token).Role)

	var existingWorkspace workspaces_dto.WorkspaceResponseDTO
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/workspaces", "Bearer "+token,
		workspaces_dto.CreateWorkspaceRequestDTO{Name: "Existing"}, http.StatusOK, &existingWorkspace)

	test_utils.MakePostRequest(t, router, "/api/v1/storages", "Bearer "+token, storages.Storage{
		WorkspaceID:  existingWorkspace.ID,
		Type:         storages.StorageTypeLocal,
		Name:         "Existing storage",
		LocalStorage: &local_storage.LocalStorage{},
	}, http.StatusOK)

	workspacesBefore := listWorkspaces(t, router, token)
	storagesBefore := listStorages(t, router, token, existingWorkspace.ID)

	users_testing.DisableMemberWorkspaceCreation(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.WorkspaceName)
	assert.Contains(t, output, workspaces_errors.ErrInsufficientPermissionsToCreateWorkspaces.Error())

	assert.Equal(t, workspacesBefore, listWorkspaces(t, router, token))
	assert.Equal(t, storagesBefore, listStorages(t, router, token, existingWorkspace.ID))
	assert.Equal(t, uuid.Nil, findWorkspaceID(t, router, token, params.WorkspaceName))
}

func Test_Seed_WhenBackendDoesNotAnswer_FailsWithDevTaskHint(t *testing.T) {
	stoppedServer := httptest.NewServer(http.NotFoundHandler())
	stoppedServer.Close()

	params := buildTestSeedParams(stoppedServer.URL)

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, "mise run dev")
}

// A full disk is unsafe to reproduce, so a stub stands in for the backend's health check.
func Test_Seed_WhenBackendReportsItselfUnhealthy_FailsWithReasonAndSendsNothingElse(t *testing.T) {
	const unhealthyReason = "more than 95% of the disk is used"

	var receivedPathsMutex sync.Mutex
	var receivedPaths []string

	stubServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPathsMutex.Lock()
		receivedPaths = append(receivedPaths, request.URL.Path)
		receivedPathsMutex.Unlock()

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(`{"status":"` + unhealthyReason + `"}`))
	}))
	t.Cleanup(stubServer.Close)

	exitCode, output := runSeedForTest(t, buildTestSeedParams(stubServer.URL))

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, unhealthyReason)

	receivedPathsMutex.Lock()
	defer receivedPathsMutex.Unlock()
	assert.Equal(t, []string{"/api/v1/system/health"}, receivedPaths)
}

type seededItems struct {
	workspaceIDs []uuid.UUID
	storageIDs   []uuid.UUID
	databaseIDs  []uuid.UUID
}

// Every test claims the instance with an account of its own first, so the seeded account
// registers as a member, the way it does on a dev instance someone already uses.
func setUpSeedTest(t *testing.T) (*gin.Engine, seedParams) {
	t.Helper()

	router := newSeedTestRouter()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	users_testing.ResetSettingsToDefaults(t.Context())

	instanceOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), instanceOwner.UserID) })

	params := buildTestSeedParams(server.URL)
	t.Cleanup(func() { removeAccountWithItems(t, router, params.Email) })

	return router, params
}

func newSeedTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	userController.RegisterProtectedRoutes(protected)
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected)
	storages.GetStorageController().RegisterRoutes(protected)
	databases.GetDatabaseController().RegisterRoutes(protected)
	backups_config_logical.GetBackupConfigController().RegisterRoutes(protected)
	backups_config_physical.GetBackupConfigController().RegisterRoutes(protected)
	backups_controllers_logical.GetBackupController().RegisterRoutes(protected)
	backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(protected)

	databases.SetupDependencies()
	backups_services.SetupDependencies()
	audit_logs.SetupDependencies()
	notifiers.SetupDependencies()
	storages.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	return router
}

func buildTestSeedParams(baseURL string) seedParams {
	env := config.GetEnv()

	logicalPort, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(err)
	}

	physicalPort, err := strconv.Atoi(env.TestPhysicalPostgres18Port)
	if err != nil {
		panic(err)
	}

	return seedParams{
		BaseURL:             baseURL,
		Email:               "seed-" + uuid.NewString() + "@example.com",
		Password:            seedPassword,
		WorkspaceName:       "Dev",
		StorageName:         "Local",
		LogicalDatabaseName: "Logical PG16",
		LogicalSource: postgresSource{
			Host:     env.TestLocalhost,
			Port:     logicalPort,
			Username: "testuser",
			Password: "testpassword",
			Database: "testdb",
		},
		WalDatabaseName: "WAL PG18",
		WalSource: postgresSource{
			Host:     env.TestLocalhost,
			Port:     physicalPort,
			Username: "testuser",
			Password: "testpassword",
		},
	}
}

func runSeedForTest(t *testing.T, params seedParams) (int, string) {
	t.Helper()

	var output bytes.Buffer
	exitCode := runSeed(t.Context(), &http.Client{Timeout: requestTimeout}, params,
		slog.New(slog.NewTextHandler(&output, nil)))

	return exitCode, output.String()
}

func signUpForTest(t *testing.T, router *gin.Engine, email, password string) string {
	t.Helper()

	var signUp users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/users/signup", "",
		users_dto.SignUpRequestDTO{Email: email, Password: password, Name: "Seed Test"},
		http.StatusOK, &signUp)

	return signUp.Token
}

func signInForTest(t *testing.T, router *gin.Engine, email, password string) string {
	t.Helper()

	var signIn users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/users/signin", "",
		users_dto.SignInRequestDTO{Email: email, Password: password}, http.StatusOK, &signIn)
	require.NotEmpty(t, signIn.Token)

	return signIn.Token
}

func getProfile(t *testing.T, router *gin.Engine, token string) users_dto.UserProfileResponseDTO {
	t.Helper()

	var profile users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/users/me", "Bearer "+token,
		http.StatusOK, &profile)

	return profile
}

func listWorkspaces(t *testing.T, router *gin.Engine, token string) []workspaces_dto.WorkspaceResponseDTO {
	t.Helper()

	var listedWorkspaces workspaces_dto.ListWorkspacesResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/workspaces", "Bearer "+token,
		http.StatusOK, &listedWorkspaces)

	return listedWorkspaces.Workspaces
}

func findWorkspaceID(t *testing.T, router *gin.Engine, token, name string) uuid.UUID {
	t.Helper()

	for _, workspace := range listWorkspaces(t, router, token) {
		if workspace.Name == name {
			return workspace.ID
		}
	}

	return uuid.Nil
}

func findStorageID(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, storage := range listStorages(t, router, token, workspaceID) {
		if storage.Name == name {
			return storage.ID
		}
	}

	return uuid.Nil
}

func findDatabaseID(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, database := range listDatabases(t, router, token, workspaceID) {
		if database.Name == name {
			return database.ID
		}
	}

	return uuid.Nil
}

func listStorages(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID) []storages.Storage {
	t.Helper()

	var listedStorages []storages.Storage
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/storages?workspace_id="+workspaceID.String(),
		"Bearer "+token, http.StatusOK, &listedStorages)

	return listedStorages
}

func listDatabases(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID) []databases.Database {
	t.Helper()

	var listedDatabases []databases.Database
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/databases?workspace_id="+workspaceID.String(),
		"Bearer "+token, http.StatusOK, &listedDatabases)

	return listedDatabases
}

func listSeededItems(t *testing.T, router *gin.Engine, token string) seededItems {
	t.Helper()

	var items seededItems

	for _, workspace := range listWorkspaces(t, router, token) {
		items.workspaceIDs = append(items.workspaceIDs, workspace.ID)

		for _, storage := range listStorages(t, router, token, workspace.ID) {
			items.storageIDs = append(items.storageIDs, storage.ID)
		}

		for _, database := range listDatabases(t, router, token, workspace.ID) {
			items.databaseIDs = append(items.databaseIDs, database.ID)
		}
	}

	return items
}

// A logical backup the seed started still writes its file, and deleting its database before it
// finishes would leave the file behind.
func waitForLogicalBackups(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	backuping_logical.WaitForBackupCompletion(t, databaseID, 0, firstBackupTimeout)
}

func removeAccountWithItems(t *testing.T, router *gin.Engine, email string) {
	t.Helper()

	ctx := context.Background()

	user, err := users_services.GetUserService().GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return
	}

	access, err := users_services.GetUserService().GenerateAccessToken(ctx, user)
	require.NoError(t, err)

	token := access.Token
	items := listSeededItems(t, router, token)

	for _, workspaceID := range items.workspaceIDs {
		for _, database := range listDatabases(t, router, token, workspaceID) {
			if database.Type == databases.DatabaseTypePostgresLogical {
				waitForLogicalBackups(t, database.ID)
			}

			test_utils.MakeDeleteRequest(t, router, "/api/v1/databases/"+database.ID.String(),
				"Bearer "+token, http.StatusNoContent)
		}

		for _, storage := range listStorages(t, router, token, workspaceID) {
			test_utils.MakeDeleteRequest(t, router, "/api/v1/storages/"+storage.ID.String(),
				"Bearer "+token, http.StatusOK)
		}

		test_utils.MakeDeleteRequest(t, router, "/api/v1/workspaces/"+workspaceID.String(),
			"Bearer "+token, http.StatusOK)
	}

	users_testing.DeleteTestUser(ctx, user.ID)
}
