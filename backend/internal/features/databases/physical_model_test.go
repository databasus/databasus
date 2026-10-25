package databases

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/tools"
)

type physicalFixture struct {
	name    string
	version tools.PostgresqlVersion
	port    func() string
}

func physicalFixtures() []physicalFixture {
	return []physicalFixture{
		{"pg17", tools.PostgresqlVersion17, func() string { return config.GetEnv().TestPhysicalPostgres17Port }},
		{"pg18", tools.PostgresqlVersion18, func() string { return config.GetEnv().TestPhysicalPostgres18Port }},
	}
}

func buildPhysicalDatabaseRequest(t *testing.T, workspaceID uuid.UUID, fx physicalFixture) Database {
	t.Helper()

	port, err := strconv.Atoi(fx.port())
	require.NoError(t, err)

	return Database{
		WorkspaceID: &workspaceID,
		Name:        "Physical DB " + uuid.New().String(),
		Type:        DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    fx.version,
			Host:       config.GetEnv().TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: postgresql_physical.BackupTypeFullOnly,
		},
	}
}

func createPhysicalDatabaseInternalAPI(
	t *testing.T,
	router *gin.Engine,
	workspaceID uuid.UUID,
	token string,
	fx physicalFixture,
) *Database {
	t.Helper()

	request := buildPhysicalDatabaseRequest(t, workspaceID, fx)

	w := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create",
		"Bearer "+token, request,
	)
	require.Equal(t, http.StatusCreated, w.Code, "create physical database failed: %s", w.Body.String())

	var database Database
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))

	return &database
}

func Test_TestConnection_OnPhysicalDatabase_DispatchesToReplicationConnection(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/"+database.ID.String()+"/test-connection",
				"Bearer "+owner.Token,
				nil,
				http.StatusOK,
			)
		})
	}
}

func Test_CreateReplicationOnlyUser_OnPhysicalDatabase_ReturnsCredentials(t *testing.T) {
	for _, fixture := range physicalFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fixture)
			defer func() {
				RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			var createReplicationOnlyUserResponse CreateReplicationOnlyUserResponse
			test_utils.MakePostRequestAndUnmarshal(
				t, router,
				"/api/v1/databases/create-replication-only-user",
				"Bearer "+owner.Token,
				database,
				http.StatusOK,
				&createReplicationOnlyUserResponse,
			)

			assert.NotEmpty(t, createReplicationOnlyUserResponse.Username)
			assert.NotEmpty(t, createReplicationOnlyUserResponse.Password)
			assert.Contains(t, createReplicationOnlyUserResponse.Username, "databasus-")
			assert.True(t, createReplicationOnlyUserResponse.IsForcedWalRotationAvailable)
		})
	}
}

func Test_TestDatabaseConnectionDirect_OnReplicationReadyCluster_ReturnsSuccess(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/test-connection-direct",
				"Bearer "+owner.Token,
				database,
				http.StatusOK,
			)
		})
	}
}

func Test_TestDatabaseConnectionDirect_WithoutPhysicalConfig_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	databaseWithoutConfig := Database{
		WorkspaceID: &workspace.ID,
		Name:        "physical db without config",
		Type:        DatabaseTypePostgresPhysical,
	}

	response := test_utils.MakePostRequest(
		t, router,
		"/api/v1/databases/test-connection-direct",
		"Bearer "+owner.Token,
		databaseWithoutConfig,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "postgresql physical config is not set")
}

func Test_CreatePhysicalDatabase_DetectsVersionFromServer_OverridingPayloadVersion(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			request := buildPhysicalDatabaseRequest(t, workspace.ID, fx)
			request.PostgresqlPhysical.Version = tools.PostgresqlVersion16

			w := workspaces_testing.MakeAPIRequest(
				router, "POST", "/api/v1/databases/create",
				"Bearer "+owner.Token, request,
			)
			require.Equal(t, http.StatusCreated, w.Code, "create failed: %s", w.Body.String())

			var database Database
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &database))
			defer func() {
				RemoveTestDatabase(t.Context(), &database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			assert.Equal(t, fx.version, database.PostgresqlPhysical.Version)
		})
	}
}

func Test_CreatePhysicalDatabase_OnUnsupportedServerVersion_ReturnsError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	port, err := strconv.Atoi(config.GetEnv().TestLogicalPostgres16Port)
	require.NoError(t, err)

	request := Database{
		WorkspaceID: &workspace.ID,
		Name:        "Physical DB " + uuid.New().String(),
		Type:        DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    tools.PostgresqlVersion18,
			Host:       config.GetEnv().TestLocalhost,
			Port:       port,
			Username:   "testuser",
			Password:   "testpassword",
			BackupType: postgresql_physical.BackupTypeFullOnly,
		},
	}

	w := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create",
		"Bearer "+owner.Token, request,
	)

	require.Equal(t, http.StatusBadRequest, w.Code, "expected rejection, got: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "17 or 18")
}

func Test_CreateReplicationOnlyUser_OnLogicalDatabase_ReturnsBadRequest(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	logicalDB := createTestDatabaseViaAPI("Logical DB", workspace.ID, owner.Token, router)
	defer func() {
		RemoveTestDatabase(t.Context(), logicalDB)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	testResp := test_utils.MakePostRequest(
		t, router,
		"/api/v1/databases/create-replication-only-user",
		"Bearer "+owner.Token,
		logicalDB,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResp.Body), "POSTGRES_PHYSICAL")
}

func Test_UpdatePhysicalDatabase_KeepsSameChildRow_WithStableIdAndDatabaseId(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			database := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)
			defer func() {
				RemoveTestDatabase(t.Context(), database)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			createdDatabase, err := getDatabaseRepository().FindByID(database.ID)
			require.NoError(t, err)
			require.NotNil(t, createdDatabase.PostgresqlPhysical)
			require.NotEqual(t, uuid.Nil, createdDatabase.PostgresqlPhysical.ID)
			require.NotNil(t, createdDatabase.PostgresqlPhysical.DatabaseID)
			assert.Equal(t, database.ID, *createdDatabase.PostgresqlPhysical.DatabaseID)

			createdDatabase.Name = "Renamed " + createdDatabase.Name
			test_utils.MakePostRequest(
				t, router,
				"/api/v1/databases/update",
				"Bearer "+owner.Token,
				createdDatabase,
				http.StatusOK,
			)

			updatedDatabase, err := getDatabaseRepository().FindByID(database.ID)
			require.NoError(t, err)
			require.NotNil(t, updatedDatabase.PostgresqlPhysical)
			assert.Equal(t,
				createdDatabase.PostgresqlPhysical.ID,
				updatedDatabase.PostgresqlPhysical.ID,
				"update must reuse the existing physical config row, not insert a second one")
			require.NotNil(t, updatedDatabase.PostgresqlPhysical.DatabaseID)
			assert.Equal(t, database.ID, *updatedDatabase.PostgresqlPhysical.DatabaseID)
			assert.Equal(t,
				createdDatabase.PostgresqlPhysical.ReplicationSlotName,
				updatedDatabase.PostgresqlPhysical.ReplicationSlotName)
		})
	}
}

func Test_CopyDatabase_OnPhysicalDatabase_DoesNotCopySystemIdentifier(t *testing.T) {
	for _, fx := range physicalFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			router := createTestRouter()
			owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			originalDatabase := createPhysicalDatabaseInternalAPI(t, router, workspace.ID, owner.Token, fx)

			originalPersistedDatabase, err := getDatabaseRepository().FindByID(originalDatabase.ID)
			require.NoError(t, err)
			require.NotNil(t, originalPersistedDatabase.PostgresqlPhysical)
			require.NotNil(t, originalPersistedDatabase.PostgresqlPhysical.SystemIdentifier,
				"original SystemIdentifier should be populated on create")
			originalSlotName := originalPersistedDatabase.PostgresqlPhysical.ReplicationSlotName

			var copiedDatabase Database
			test_utils.MakePostRequestAndUnmarshal(
				t, router,
				"/api/v1/databases/"+originalDatabase.ID.String()+"/copy",
				"Bearer "+owner.Token,
				nil,
				http.StatusCreated,
				&copiedDatabase,
			)
			defer func() {
				RemoveTestDatabase(t.Context(), &copiedDatabase)
				RemoveTestDatabase(t.Context(), originalDatabase)
				workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
			}()

			copiedPersistedDatabase, err := getDatabaseRepository().FindByID(copiedDatabase.ID)
			require.NoError(t, err)
			require.NotNil(t, copiedPersistedDatabase.PostgresqlPhysical)
			assert.NotEqual(t, originalDatabase.ID, copiedDatabase.ID)
			assert.NotEqual(t, originalSlotName, copiedPersistedDatabase.PostgresqlPhysical.ReplicationSlotName,
				"copied database must have a fresh replication slot name")
		})
	}
}
