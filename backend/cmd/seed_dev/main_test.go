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

	contract "databasus-backend/api"
	"databasus-backend/internal/config"
	"databasus-backend/internal/engines/clienttools"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	databases_bridge "databasus-backend/internal/features/databases"
	storages_bridge "databasus-backend/internal/features/storages"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	users_bridge "databasus-backend/internal/features/users"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
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

	token := signInForTest(t, router, params.Email, seedPassword).Token
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

	token := signInForTest(t, router, params.Email, seedPassword).Token
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

	token := signInForTest(t, router, params.Email, seedPassword).Token
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
	sessionBefore := signInForTest(t, router, params.Email, otherPassword)
	profileBefore := getProfile(t, router, sessionBefore.UserId, sessionBefore.Token)

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.Email)
	assert.Contains(t, output, "password is incorrect")

	sessionAfter := signInForTest(t, router, params.Email, otherPassword)
	assert.Equal(t, profileBefore, getProfile(t, router, sessionAfter.UserId, sessionAfter.Token))
	assert.Empty(t, listSeededItems(t, router, sessionAfter.Token).workspaceIDs)
}

func Test_Seed_WhenAnotherAccountExists_LeavesThatAccountUnchanged(t *testing.T) {
	router, params := setUpSeedTest(t)

	otherEmail := "seed-other-" + uuid.NewString() + "@example.com"
	signUpForTest(t, router, otherEmail, otherPassword)
	t.Cleanup(func() { removeAccountWithItems(t, router, otherEmail) })

	otherSessionBefore := signInForTest(t, router, otherEmail, otherPassword)
	otherProfileBefore := getProfile(t, router, otherSessionBefore.UserId, otherSessionBefore.Token)

	exitCode, output := runSeedForTest(t, params)
	require.Equal(t, 0, exitCode, output)

	otherSession := signInForTest(t, router, otherEmail, otherPassword)
	assert.Equal(t, otherProfileBefore, getProfile(t, router, otherSession.UserId, otherSession.Token))
	assert.Empty(t, listSeededItems(t, router, otherSession.Token).workspaceIDs)
}

func Test_Seed_WhenTwoFactorSignInIsRequired_FailsAndCreatesNothing(t *testing.T) {
	router, params := setUpSeedTest(t)

	token := signUpForTest(t, router, params.Email, seedPassword).Token

	users_testing.EnableTwoFactorAuth(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.Email)
	assert.Contains(t, output, "two-factor sign-in")
	assert.Empty(t, listSeededItems(t, router, token).workspaceIDs)
}

func Test_Seed_WhenMemberMayNotCreateWorkspaces_FailsNamingWorkspace(t *testing.T) {
	router, params := setUpSeedTest(t)

	session := signUpForTest(t, router, params.Email, seedPassword)
	token := session.Token
	require.Equal(t, apiclient.UserRoleMEMBER, getProfile(t, router, session.UserId, session.Token).Role)

	var existingWorkspace apiclient.Workspace
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/workspaces", "Bearer "+token,
		apiclient.WorkspaceCreate{Name: "Existing"}, http.StatusCreated, &existingWorkspace)

	test_utils.MakePostRequest(t, router, "/api/v1/storages", "Bearer "+token, map[string]any{
		"workspaceId": existingWorkspace.Id,
		"name":        "Existing storage",
		"settings":    map[string]any{"type": "LOCAL"},
	}, http.StatusCreated)

	workspacesBefore := listWorkspaces(t, router, token)
	storagesBefore := listStorages(t, router, token, existingWorkspace.Id)

	users_testing.DisableMemberWorkspaceCreation(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

	exitCode, output := runSeedForTest(t, params)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, output, params.WorkspaceName)
	assert.Contains(t, output, "the instance does not allow this user to create workspaces")

	assert.Equal(t, workspacesBefore, listWorkspaces(t, router, token))
	assert.Equal(t, storagesBefore, listStorages(t, router, token, existingWorkspace.Id))
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

	instanceOwner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	t.Cleanup(func() { users_testing.DeleteTestUser(context.Background(), instanceOwner.UserID) })

	params := buildTestSeedParams(server.URL)
	t.Cleanup(func() { removeAccountWithItems(t, router, params.Email) })

	return router, params
}

func newSeedTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	contractGuard, err := httpserver.RequireContractOperation(contract.OpenAPI, logger.GetLogger())
	if err != nil {
		panic(err)
	}

	usersModule := users_bridge.NewModuleWithoutApp(acceptingMailer{})
	users_bridge.Install(usersModule)
	workspacesModule := workspaces_bridge.NewModuleWithoutApp(usersModule)
	workspaces_bridge.Install(workspacesModule)

	contractRoutes := router.Group("/api/v1", contractGuard, usersModule.GetAccessMiddleware())
	errorHooks := apierror.NewHooks(apierror.Deps{Logger: logger.GetLogger()})
	usersModule.RegisterRoutes(contractRoutes, errorHooks)
	workspacesModule.RegisterRoutes(contractRoutes, errorHooks)
	storages_bridge.GetModule().RegisterRoutes(contractRoutes, errorHooks)
	databases_bridge.GetModule().RegisterRoutes(contractRoutes, errorHooks)

	system_healthcheck.InstallClientToolsChecker(clienttools.NewChecker(nil))
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(users_testing.GetSessionMiddleware())

	backups_config_logical.GetBackupConfigController().RegisterRoutes(protected)
	backups_config_physical.GetBackupConfigController().RegisterRoutes(protected)
	backups_controllers_logical.GetBackupController().RegisterRoutes(protected)
	backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(protected)

	databases_bridge.SetupDependencies()
	backups_services.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()
	backups_services.SubscribeToDatabaseEventsWithoutApp()
	backuping_physical.SubscribeToDatabaseDeletingWithoutApp()

	return router
}

// acceptingMailer is a mail server that accepts every message, so an instance that requires
// two-factor sign-in answers the seed with a code request instead of a refusal to send the code.
type acceptingMailer struct{}

func (acceptingMailer) IsConfigured() bool {
	return true
}

func (acceptingMailer) SendEmail(context.Context, string, string, string) error {
	return nil
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

func signUpForTest(t *testing.T, router *gin.Engine, email, password string) apiclient.Session {
	t.Helper()

	var signUp apiclient.Session
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/users", "",
		signUpRequest{Email: email, Password: password, Name: "Seed Test"},
		http.StatusCreated, &signUp)

	return signUp
}

func signInForTest(t *testing.T, router *gin.Engine, email, password string) apiclient.SignInCompleted {
	t.Helper()

	var signIn apiclient.SignInCompleted
	test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/sessions", "",
		signInRequest{Email: email, Password: password}, http.StatusOK, &signIn)
	require.NotEmpty(t, signIn.Token)

	return signIn
}

func getProfile(t *testing.T, router *gin.Engine, userID uuid.UUID, token string) apiclient.User {
	t.Helper()

	var profile apiclient.User
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/users/"+userID.String(), "Bearer "+token,
		http.StatusOK, &profile)

	return profile
}

func listWorkspaces(t *testing.T, router *gin.Engine, token string) []apiclient.Workspace {
	t.Helper()

	var listedWorkspaces apiclient.WorkspacePage
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/workspaces?limit=100", "Bearer "+token,
		http.StatusOK, &listedWorkspaces)

	return listedWorkspaces.Items
}

func findWorkspaceID(t *testing.T, router *gin.Engine, token, name string) uuid.UUID {
	t.Helper()

	for _, workspace := range listWorkspaces(t, router, token) {
		if workspace.Name == name {
			return workspace.Id
		}
	}

	return uuid.Nil
}

func findStorageID(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, storage := range listStorages(t, router, token, workspaceID) {
		if storage.Name == name {
			return storage.Id
		}
	}

	return uuid.Nil
}

func findDatabaseID(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, database := range listDatabases(t, router, token, workspaceID) {
		if database.Name == name {
			return database.Id
		}
	}

	return uuid.Nil
}

// listStorages reads one page of 100, more than a seeded workspace holds.
func listStorages(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID) []apiclient.Storage {
	t.Helper()

	var storagePage apiclient.StoragePage
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/storages?limit=100&workspaceId="+workspaceID.String(),
		"Bearer "+token, http.StatusOK, &storagePage)

	return storagePage.Items
}

// listDatabases reads one page of 100, more than a seeded workspace holds.
func listDatabases(t *testing.T, router *gin.Engine, token string, workspaceID uuid.UUID) []apiclient.Database {
	t.Helper()

	var databasePage apiclient.DatabasePage
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/databases?limit=100&workspaceId="+workspaceID.String(),
		"Bearer "+token, http.StatusOK, &databasePage)

	return databasePage.Items
}

func listSeededItems(t *testing.T, router *gin.Engine, token string) seededItems {
	t.Helper()

	var items seededItems

	for _, workspace := range listWorkspaces(t, router, token) {
		items.workspaceIDs = append(items.workspaceIDs, workspace.Id)

		for _, storage := range listStorages(t, router, token, workspace.Id) {
			items.storageIDs = append(items.storageIDs, storage.Id)
		}

		for _, database := range listDatabases(t, router, token, workspace.Id) {
			items.databaseIDs = append(items.databaseIDs, database.Id)
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

	var userIDs []uuid.UUID
	require.NoError(t, storage.GetDb().WithContext(ctx).
		Raw("SELECT id FROM users WHERE email = ?", email).Scan(&userIDs).Error)
	if len(userIDs) == 0 {
		return
	}

	userID := userIDs[0]
	token := users_testing.IssueTestToken(ctx, userID)
	items := listSeededItems(t, router, token)

	for _, workspaceID := range items.workspaceIDs {
		for _, database := range listDatabases(t, router, token, workspaceID) {
			databaseType, err := database.Settings.Discriminator()
			require.NoError(t, err)

			if databaseType == string(apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL) {
				waitForLogicalBackups(t, database.Id)
			}

			test_utils.MakeDeleteRequest(t, router, "/api/v1/databases/"+database.Id.String(),
				"Bearer "+token, http.StatusNoContent)
		}

		for _, storage := range listStorages(t, router, token, workspaceID) {
			test_utils.MakeDeleteRequest(t, router, "/api/v1/storages/"+storage.Id.String(),
				"Bearer "+token, http.StatusNoContent)
		}

		test_utils.MakeDeleteRequest(t, router, "/api/v1/workspaces/"+workspaceID.String(),
			"Bearer "+token, http.StatusNoContent)
	}

	users_testing.DeleteTestUser(ctx, userID)
}
