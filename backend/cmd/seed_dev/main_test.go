package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	seedPassword  = "Q1234567"
	otherPassword = "Other-password-1"
)

func TestSeed(t *testing.T) {
	t.Parallel()

	t.Run("creates workspace storage and configured backups when the account is new", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

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

		logicalConfig := getBackupConfig(t, router, token, logicalDatabaseID)
		logicalSettings, err := logicalConfig.Settings.AsLogicalBackupSettings()
		require.NoError(t, err)

		assert.True(t, logicalConfig.IsBackupsEnabled)
		assert.Equal(t, storageID, *logicalConfig.StorageId)
		assert.Equal(t, apiclient.LogicalRetentionPolicyCOUNT, logicalSettings.Retention.Policy)
		assert.Equal(t, 3, logicalSettings.Retention.Count)

		var logicalBackups apiclient.LogicalBackupPage
		backupResponse, err := router.NewSessionClient(t, token).
			ListLogicalBackupsWithResponse(t.Context(), &apiclient.ListLogicalBackupsParams{DatabaseId: logicalDatabaseID})
		require.NoError(t, err)
		require.NotNil(t, backupResponse.JSON200)
		logicalBackups = *backupResponse.JSON200
		assert.NotEmpty(t, logicalBackups.Items, "the first logical backup was not started")

		walConfig := getBackupConfig(t, router, token, walDatabaseID)
		walSettings, err := walConfig.Settings.AsPhysicalBackupSettings()
		require.NoError(t, err)

		assert.True(t, walConfig.IsBackupsEnabled)
		assert.Equal(t, storageID, *walConfig.StorageId)
		assert.Equal(t, apiclient.PhysicalRetentionPolicyCHAINS, walSettings.Retention.Policy)
		assert.Equal(t, 2, walSettings.Retention.ChainsCount)

		var forceFullRequests int
		router.QuerySQL(
			t,
			&forceFullRequests,
			"SELECT COUNT(*) FROM backups_configs_physical WHERE database_id = ? AND force_full_requested_at IS NOT NULL",
			walDatabaseID,
		)
		assert.Equal(t, 1, forceFullRequests, "the first WAL-streaming backup was not requested")
	})

	t.Run("preserves existing items and edited backup settings when run twice", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

		exitCode, output := runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		token := signInForTest(t, router, params.Email, seedPassword).Token
		workspaceID := findWorkspaceID(t, router, token, params.WorkspaceName)
		logicalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName)
		waitForLogicalBackups(t, seedDatabase{app: router, token: token, databaseID: logicalDatabaseID})

		editedConfig := getBackupConfig(t, router, token, logicalDatabaseID)
		editedSettings, err := editedConfig.Settings.AsLogicalBackupSettings()
		require.NoError(t, err)

		editedSettings.Retention.Count = 9
		require.NoError(t, editedConfig.Settings.FromLogicalBackupSettings(editedSettings))
		response, err := router.NewSessionClient(t, token).
			UpdateBackupConfigWithResponse(t.Context(), logicalDatabaseID, apiclient.BackupConfigInput{
				IsBackupsEnabled: false,
				StorageId:        editedConfig.StorageId,
				Encryption:       editedConfig.Encryption,
				Settings:         editedConfig.Settings,
			})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode())

		itemsBefore := listSeededItems(t, router, token)

		exitCode, output = runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		assert.Equal(t, itemsBefore, listSeededItems(t, router, token))

		configAfter := getBackupConfig(t, router, token, logicalDatabaseID)
		settingsAfter, err := configAfter.Settings.AsLogicalBackupSettings()
		require.NoError(t, err)

		assert.Equal(t, 9, settingsAfter.Retention.Count)
		assert.False(t, configAfter.IsBackupsEnabled)
	})

	t.Run("recreates only the deleted database when seeded again", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

		exitCode, output := runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		token := signInForTest(t, router, params.Email, seedPassword).Token
		workspaceID := findWorkspaceID(t, router, token, params.WorkspaceName)
		storageID := findStorageID(t, router, token, workspaceID, params.StorageName)
		logicalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName)
		deletedWalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.WalDatabaseName)

		response, err := router.NewSessionClient(t, token).DeleteDatabaseWithResponse(t.Context(), deletedWalDatabaseID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode())

		exitCode, output = runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		recreatedWalDatabaseID := findDatabaseID(t, router, token, workspaceID, params.WalDatabaseName)
		require.NotEqual(t, uuid.Nil, recreatedWalDatabaseID)
		assert.NotEqual(t, deletedWalDatabaseID, recreatedWalDatabaseID)

		assert.True(t, getBackupConfig(t, router, token, recreatedWalDatabaseID).IsBackupsEnabled)

		assert.Equal(t, workspaceID, findWorkspaceID(t, router, token, params.WorkspaceName))
		assert.Equal(t, storageID, findStorageID(t, router, token, workspaceID, params.StorageName))
		assert.Equal(t, logicalDatabaseID, findDatabaseID(t, router, token, workspaceID, params.LogicalDatabaseName))
		assert.Len(t, listSeededItems(t, router, token).databaseIDs, 2)
	})

	t.Run("preserves the account when its password differs", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

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
	})

	t.Run("preserves another account when seeding a new account", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

		otherEmail := "seed-other-" + uuid.NewString() + "@example.com"
		signUpForTest(t, router, otherEmail, otherPassword)

		otherSessionBefore := signInForTest(t, router, otherEmail, otherPassword)
		otherProfileBefore := getProfile(t, router, otherSessionBefore.UserId, otherSessionBefore.Token)

		exitCode, output := runSeedForTest(t, params)
		require.Equal(t, 0, exitCode, output)

		otherSession := signInForTest(t, router, otherEmail, otherPassword)
		assert.Equal(t, otherProfileBefore, getProfile(t, router, otherSession.UserId, otherSession.Token))
		assert.Empty(t, listSeededItems(t, router, otherSession.Token).workspaceIDs)
	})

	t.Run("creates nothing when two factor sign in is required", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

		token := signUpForTest(t, router, params.Email, seedPassword).Token

		updateSeedSettings(t, fixture, seedSettings{
			isTwoFactorRequired: true, isWorkspaceCreationAllowed: true,
		})

		exitCode, output := runSeedForTest(t, params)

		assert.Equal(t, 1, exitCode)
		assert.Contains(t, output, params.Email)
		assert.Contains(t, output, "two-factor sign-in")
		assert.Empty(t, listSeededItems(t, router, token).workspaceIDs)
	})

	t.Run("names the workspace when its creation is forbidden", func(t *testing.T) {
		t.Parallel()

		fixture, params := setUpSeedTest(t)
		router := fixture.router

		session := signUpForTest(t, router, params.Email, seedPassword)
		token := session.Token
		require.Equal(t, apiclient.UserRoleMEMBER, getProfile(t, router, session.UserId, session.Token).Role)

		existingSession := apptest.Session{
			Client: router.NewSessionClient(t, token),
			UserID: session.UserId,
			Token:  token,
		}
		existingWorkspace := apptest.CreateWorkspace(t, existingSession, "Existing")
		router.CreateLocalStorage(t, existingSession, existingWorkspace.Id)
		workspacesBefore := listWorkspaces(t, router, token)
		storagesBefore := listStorages(t, router, token, existingWorkspace.Id)

		updateSeedSettings(t, fixture, seedSettings{
			isTwoFactorRequired: false, isWorkspaceCreationAllowed: false,
		})

		exitCode, output := runSeedForTest(t, params)

		assert.Equal(t, 1, exitCode)
		assert.Contains(t, output, params.WorkspaceName)
		assert.Contains(t, output, "the instance does not allow this user to create workspaces")

		assert.Equal(t, workspacesBefore, listWorkspaces(t, router, token))
		assert.Equal(t, storagesBefore, listStorages(t, router, token, existingWorkspace.Id))
		assert.Equal(t, uuid.Nil, findWorkspaceID(t, router, token, params.WorkspaceName))
	})

	t.Run("suggests the dev task when the backend does not answer", func(t *testing.T) {
		t.Parallel()

		stoppedServer := httptest.NewServer(http.NotFoundHandler())
		stoppedServer.Close()

		params := buildTestSeedParams(stoppedServer.URL)

		exitCode, output := runSeedForTest(t, params)

		assert.Equal(t, 1, exitCode)
		assert.Contains(t, output, "mise run run")
	})

	t.Run("reports the reason without other requests when the backend is unhealthy", func(t *testing.T) {
		t.Parallel()

		// A full disk is unsafe to reproduce, so a stub stands in for the backend health check.
		const unhealthyReason = "more than 95% of the disk is used"

		var receivedPathsMutex sync.Mutex
		var receivedPaths []string

		stubServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			receivedPathsMutex.Lock()
			receivedPaths = append(receivedPaths, request.URL.Path)
			receivedPathsMutex.Unlock()

			writer.Header().Set("Content-Type", "application/problem+json")
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte(`{"status":503,"code":"instance_unhealthy","detail":"` + unhealthyReason + `"}`))
		}))
		t.Cleanup(stubServer.Close)

		exitCode, output := runSeedForTest(t, buildTestSeedParams(stubServer.URL))

		assert.Equal(t, 1, exitCode)
		assert.Contains(t, output, unhealthyReason)

		receivedPathsMutex.Lock()
		defer receivedPathsMutex.Unlock()
		assert.Equal(t, []string{"/api/v1/system/health"}, receivedPaths)
	})
}

type seededItems struct {
	workspaceIDs []uuid.UUID
	storageIDs   []uuid.UUID
	databaseIDs  []uuid.UUID
}

type seedFixture struct {
	router *apptest.App
	admin  apptest.Session
}

func setUpSeedTest(t *testing.T) (seedFixture, seedParams) {
	t.Helper()
	logicalSource := containers.StartPostgres(t, "postgres:16")
	walSource := containers.StartPhysicalPostgres(t, "postgres:18")

	testApp := apptest.New(t)
	admin := testApp.CreateAdmin(t)

	response, err := admin.Client.UpdateInstanceSettingsWithResponse(t.Context(), apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations: true, IsAllowMemberInvitations: true, IsMemberAllowedToCreateWorkspaces: true,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())

	params := buildTestSeedParams(testApp.URL)
	params.LogicalSource = postgresSource{
		Host:     logicalSource.Host,
		Port:     logicalSource.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
		Database: containers.PostgresDatabase,
	}
	params.WalSource = postgresSource{
		Host:     walSource.Host,
		Port:     walSource.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
	}

	return seedFixture{router: testApp, admin: admin}, params
}

type seedSettings struct {
	isTwoFactorRequired        bool
	isWorkspaceCreationAllowed bool
}

func updateSeedSettings(t *testing.T, fixture seedFixture, settings seedSettings) {
	t.Helper()
	response, err := fixture.admin.Client.UpdateInstanceSettingsWithResponse(
		t.Context(),
		apiclient.InstanceSettingsUpdate{
			IsAllowExternalRegistrations:      true,
			IsAllowMemberInvitations:          true,
			IsMemberAllowedToCreateWorkspaces: settings.isWorkspaceCreationAllowed,
			IsTwoFactorAuthRequired:           settings.isTwoFactorRequired,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
}

func buildTestSeedParams(baseURL string) seedParams {
	return seedParams{
		BaseURL:             baseURL,
		Email:               "seed-" + uuid.NewString() + "@example.com",
		Password:            seedPassword,
		WorkspaceName:       "Dev",
		StorageName:         "Local",
		LogicalDatabaseName: "Logical PG16",
		WalDatabaseName:     "WAL PG18",
	}
}

func runSeedForTest(t *testing.T, params seedParams) (int, string) {
	t.Helper()

	var output bytes.Buffer
	exitCode := runSeed(t.Context(), &http.Client{Timeout: requestTimeout}, params,
		slog.New(slog.NewTextHandler(&output, nil)))

	return exitCode, output.String()
}

func signUpForTest(t *testing.T, router *apptest.App, email, password string) apiclient.Session {
	t.Helper()

	response, err := router.NewClient(t).
		SignUpWithResponse(t.Context(), apiclient.SignUpRequest{Email: openapi_types.Email(email), Password: new(password), Name: "Seed Test"})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode())
	return *response.JSON201
}

func signInForTest(t *testing.T, router *apptest.App, email, password string) apiclient.SignInCompleted {
	t.Helper()
	response, err := router.NewClient(t).
		SignInWithResponse(t.Context(), apiclient.SignInRequest{Email: email, Password: new(password)})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	session, err := response.JSON200.AsSignInCompleted()
	require.NoError(t, err)
	require.NotEmpty(t, session.Token)
	return session
}

func getProfile(t *testing.T, router *apptest.App, userID uuid.UUID, token string) apiclient.User {
	t.Helper()
	response, err := router.NewSessionClient(t, token).GetUserWithResponse(t.Context(), userID)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return *response.JSON200
}

func listWorkspaces(t *testing.T, router *apptest.App, token string) []apiclient.Workspace {
	t.Helper()
	response, err := router.NewSessionClient(t, token).
		ListWorkspacesWithResponse(t.Context(), &apiclient.ListWorkspacesParams{Limit: new(100)})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return response.JSON200.Items
}

func findWorkspaceID(t *testing.T, router *apptest.App, token, name string) uuid.UUID {
	t.Helper()

	for _, workspace := range listWorkspaces(t, router, token) {
		if workspace.Name == name {
			return workspace.Id
		}
	}

	return uuid.Nil
}

func findStorageID(t *testing.T, router *apptest.App, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, storage := range listStorages(t, router, token, workspaceID) {
		if storage.Name == name {
			return storage.Id
		}
	}

	return uuid.Nil
}

func findDatabaseID(t *testing.T, router *apptest.App, token string, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	for _, database := range listDatabases(t, router, token, workspaceID) {
		if database.Name == name {
			return database.Id
		}
	}

	return uuid.Nil
}

func listStorages(t *testing.T, router *apptest.App, token string, workspaceID uuid.UUID) []apiclient.Storage {
	t.Helper()
	response, err := router.NewSessionClient(t, token).
		ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{WorkspaceId: workspaceID, Limit: new(100)})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return response.JSON200.Items
}

func listDatabases(t *testing.T, router *apptest.App, token string, workspaceID uuid.UUID) []apiclient.Database {
	t.Helper()
	response, err := router.NewSessionClient(t, token).
		ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{WorkspaceId: workspaceID, Limit: new(100)})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return response.JSON200.Items
}

func listSeededItems(t *testing.T, router *apptest.App, token string) seededItems {
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

type seedDatabase struct {
	app        *apptest.App
	token      string
	databaseID uuid.UUID
}

func waitForLogicalBackups(t *testing.T, database seedDatabase) {
	t.Helper()
	testwait.Until(t, "logical backups finish", func(_ context.Context) error {
		response, err := database.app.NewSessionClient(t, database.token).
			ListLogicalBackupsWithResponse(t.Context(), &apiclient.ListLogicalBackupsParams{DatabaseId: database.databaseID})
		require.NoError(t, err)
		require.NotNil(t, response.JSON200)
		for _, backup := range response.JSON200.Items {
			if backup.Status == apiclient.LogicalBackupStatusINPROGRESS {
				return fmt.Errorf("backup %s is running", backup.Id)
			}
		}
		return nil
	})
}

func getBackupConfig(t *testing.T, router *apptest.App, token string, databaseID uuid.UUID) apiclient.BackupConfig {
	t.Helper()
	response, err := router.NewSessionClient(t, token).GetBackupConfigWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200)
	return *response.JSON200
}

func TestSeedOverlap(t *testing.T) {
	t.Parallel()
	t.Run("creates both configured backups when seeds overlap", func(t *testing.T) {
		t.Parallel()

		firstFixture, firstParams := setUpSeedTest(t)
		secondFixture, secondParams := setUpSeedTest(t)
		firstParams.Email = "same-seed@example.com"
		secondParams.Email = firstParams.Email
		require.NotEqual(t, firstParams.LogicalSource.Port, secondParams.LogicalSource.Port)
		require.NotEqual(t, firstParams.WalSource.Port, secondParams.WalSource.Port)

		var seeds sync.WaitGroup
		for _, seed := range []struct {
			fixture seedFixture
			params  seedParams
		}{{firstFixture, firstParams}, {secondFixture, secondParams}} {
			seeds.Go(func() {
				exitCode, output := runSeedForTest(t, seed.params)
				assert.Equal(t, 0, exitCode, output)
			})
		}
		seeds.Wait()

		for _, seed := range []struct {
			fixture seedFixture
			params  seedParams
		}{{firstFixture, firstParams}, {secondFixture, secondParams}} {
			session := signInForTest(t, seed.fixture.router, seed.params.Email, seedPassword)
			workspaceID := findWorkspaceID(t, seed.fixture.router, session.Token, seed.params.WorkspaceName)
			logicalID := findDatabaseID(
				t,
				seed.fixture.router,
				session.Token,
				workspaceID,
				seed.params.LogicalDatabaseName,
			)
			physicalID := findDatabaseID(
				t,
				seed.fixture.router,
				session.Token,
				workspaceID,
				seed.params.WalDatabaseName,
			)

			require.NotEqual(t, uuid.Nil, logicalID)
			require.NotEqual(t, uuid.Nil, physicalID)

			waitForLogicalBackups(
				t,
				seedDatabase{app: seed.fixture.router, token: session.Token, databaseID: logicalID},
			)

			response, err := seed.fixture.router.NewSessionClient(t, session.Token).
				ListLogicalBackupsWithResponse(t.Context(), &apiclient.ListLogicalBackupsParams{DatabaseId: logicalID})
			require.NoError(t, err)
			require.NotNil(t, response.JSON200)
			require.NotEmpty(t, response.JSON200.Items)
			assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, response.JSON200.Items[0].Status)

			var requestCount int
			seed.fixture.router.QuerySQL(
				t,
				&requestCount,
				"SELECT COUNT(*) FROM backups_configs_physical WHERE database_id = ? AND force_full_requested_at IS NOT NULL",
				physicalID,
			)
			assert.Equal(t, 1, requestCount)
		}
	})
}
