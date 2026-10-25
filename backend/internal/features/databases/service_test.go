package databases

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
)

// The healthcheck path runs without an end-user principal, so the embedded-target refusal inside
// OpenTunnel is the only thing keeping a persisted row from probing Databasus's own cluster.
func Test_ProbeDatabaseLiveness_WithEmbeddedPhysicalTarget_ReturnsError(t *testing.T) {
	database := &Database{
		Type: DatabaseTypePostgresPhysical,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Host:     "/tmp",
			Port:     5437,
			Username: "postgres",
			Password: "stored-password",
		},
	}

	err := GetDatabaseService().ProbeDatabaseLiveness(t.Context(), database)

	require.ErrorContains(t, err, "backing up Databasus internal PostgreSQL cluster is not allowed")
}

// The source cluster publishes no port, so a passing probe proves it went through the bastion.
func Test_ProbeDatabaseLiveness_WhenSourceIsBehindSshBastion_Succeeds(t *testing.T) {
	topology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")

	physicalConfig := GetTestPhysicalPostgresConfigWithType(
		topology.Database.Host, topology.Database.Port, "17", postgresql_physical.BackupTypeFullOnly,
	)
	physicalConfig.SshTunnel = bastion.GetTunnelConfig(topology)

	database := &Database{
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: physicalConfig,
	}

	err := GetDatabaseService().ProbeDatabaseLiveness(t.Context(), database)

	assert.NoError(t, err)
}

type statusUpdateFixture struct {
	router    *gin.Engine
	ownerAuth string
	database  *Database
	conn      *pgx.Conn
	roleName  string
	isRemoved bool
}

// The user changes the connection password through the API, which tests the connection, so the
// fixture logs in as a role of its own whose password the test may change.
func createStatusUpdateFixture(t *testing.T) *statusUpdateFixture {
	t.Helper()

	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Status Updates", owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router) })

	postgresConfig := getTestPostgresConfig()

	conn, err := pgx.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user=testuser password=testpassword dbname=testdb sslmode=disable",
		postgresConfig.Host, postgresConfig.Port,
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	roleName := "databasus-status-" + uuid.New().String()[:8]
	_, err = conn.Exec(t.Context(), fmt.Sprintf(`CREATE ROLE "%s" WITH LOGIN PASSWORD 'first-password'`, roleName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, roleName))
	})

	request := newTestPostgresqlLogicalDatabaseRequest("Original Name", workspace.ID)
	request.PostgresqlLogical.Username = roleName
	request.PostgresqlLogical.Password = "first-password"

	var createdDatabase Database
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/databases/create", "Bearer "+owner.Token,
		request, http.StatusCreated, &createdDatabase,
	)

	fixture := &statusUpdateFixture{
		router:    router,
		ownerAuth: "Bearer " + owner.Token,
		database:  &createdDatabase,
		conn:      conn,
		roleName:  roleName,
	}
	t.Cleanup(func() {
		if !fixture.isRemoved {
			RemoveTestDatabase(t.Context(), &createdDatabase)
		}
	})

	return fixture
}

func (f *statusUpdateFixture) saveNewNameAndPassword(t *testing.T) {
	t.Helper()

	_, err := f.conn.Exec(t.Context(), fmt.Sprintf(`ALTER ROLE "%s" WITH PASSWORD 'second-password'`, f.roleName))
	require.NoError(t, err)

	var readDatabase Database
	test_utils.MakeGetRequestAndUnmarshal(
		t, f.router, "/api/v1/databases/"+f.database.ID.String(), f.ownerAuth, http.StatusOK, &readDatabase,
	)

	readDatabase.Name = "Renamed By User"
	readDatabase.PostgresqlLogical.Password = "second-password"

	test_utils.MakePostRequest(t, f.router, "/api/v1/databases/update", f.ownerAuth, readDatabase, http.StatusOK)
}

func (f *statusUpdateFixture) assertUserSettingsSurvived(t *testing.T) *Database {
	t.Helper()

	storedDatabase, err := GetDatabaseService().GetDatabaseByID(f.database.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed By User", storedDatabase.Name)

	storedPassword, err := encryption.GetFieldEncryptor().Decrypt(storedDatabase.PostgresqlLogical.Password)
	require.NoError(t, err)
	assert.Equal(t, "second-password", storedPassword)

	return storedDatabase
}

func Test_SetLastBackupTime_AfterUserSavedNewSettings_KeepsNewSettings(t *testing.T) {
	fixture := createStatusUpdateFixture(t)
	backupTime := time.Now().UTC().Truncate(time.Second)

	fixture.saveNewNameAndPassword(t)
	require.NoError(t, GetDatabaseService().SetLastBackupTime(fixture.database.ID, backupTime))

	storedDatabase := fixture.assertUserSettingsSurvived(t)
	require.NotNil(t, storedDatabase.LastBackupTime)
	assert.True(t, backupTime.Equal(*storedDatabase.LastBackupTime))
}

func Test_SetBackupError_AfterUserSavedNewSettings_KeepsNewSettings(t *testing.T) {
	fixture := createStatusUpdateFixture(t)

	fixture.saveNewNameAndPassword(t)
	require.NoError(t, GetDatabaseService().SetBackupError(fixture.database.ID, "pg_dump failed"))

	storedDatabase := fixture.assertUserSettingsSurvived(t)
	require.NotNil(t, storedDatabase.LastBackupErrorMessage)
	assert.Equal(t, "pg_dump failed", *storedDatabase.LastBackupErrorMessage)
}

func Test_SetHealthStatus_AfterUserSavedNewSettings_KeepsNewSettings(t *testing.T) {
	fixture := createStatusUpdateFixture(t)

	fixture.saveNewNameAndPassword(t)
	require.NoError(t, GetDatabaseService().SetHealthStatus(fixture.database.ID, new(HealthStatusUnavailable)))

	storedDatabase := fixture.assertUserSettingsSurvived(t)
	require.NotNil(t, storedDatabase.HealthStatus)
	assert.Equal(t, HealthStatusUnavailable, *storedDatabase.HealthStatus)
}

func Test_SetLastBackupTime_WhenDatabaseShowsBackupError_ClearsError(t *testing.T) {
	fixture := createStatusUpdateFixture(t)
	backupTime := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, GetDatabaseService().SetBackupError(fixture.database.ID, "pg_dump failed"))
	require.NoError(t, GetDatabaseService().SetLastBackupTime(fixture.database.ID, backupTime))

	storedDatabase, err := GetDatabaseService().GetDatabaseByID(fixture.database.ID)
	require.NoError(t, err)
	require.NotNil(t, storedDatabase.LastBackupTime)
	assert.True(t, backupTime.Equal(*storedDatabase.LastBackupTime))
	assert.Nil(t, storedDatabase.LastBackupErrorMessage)
	assert.Equal(t, "Original Name", storedDatabase.Name)
}

func Test_StatusSetters_WhenDatabaseDeleted_ReturnNotFoundError(t *testing.T) {
	fixture := createStatusUpdateFixture(t)

	RemoveTestDatabase(t.Context(), fixture.database)
	fixture.isRemoved = true

	databaseService := GetDatabaseService()
	assert.ErrorIs(t, databaseService.SetLastBackupTime(fixture.database.ID, time.Now().UTC()), gorm.ErrRecordNotFound)
	assert.ErrorIs(t, databaseService.SetBackupError(fixture.database.ID, "pg_dump failed"), gorm.ErrRecordNotFound)
	assert.ErrorIs(
		t,
		databaseService.SetHealthStatus(fixture.database.ID, new(HealthStatusAvailable)),
		gorm.ErrRecordNotFound,
	)
}
