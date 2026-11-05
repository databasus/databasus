package backups_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// team is an instance with a workspace owned by a user who is not an instance admin.
type team struct {
	app       *apptest.App
	admin     apptest.Session
	owner     apptest.Session
	workspace apiclient.Workspace
}

func newTeam(t *testing.T) team {
	t.Helper()

	app := apptest.New(t)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)

	return team{
		app:       app,
		admin:     admin,
		owner:     owner,
		workspace: apptest.CreateWorkspace(t, owner, "Team "+uuid.NewString()[:8]),
	}
}

func (tm team) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()

	user := tm.app.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, tm.owner, tm.workspace.Id, user.Email, role)

	return user
}

func (tm team) createWorkspace(t *testing.T) apiclient.Workspace {
	t.Helper()

	return apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
}

func (tm team) insertLogicalDatabase(t *testing.T) uuid.UUID {
	t.Helper()

	return tm.app.InsertDatabase(t, tm.workspace.Id)
}

func (tm team) createLocalStorage(t *testing.T) uuid.UUID {
	t.Helper()

	return tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
}

func (tm team) createPhysicalDatabase(
	t *testing.T,
	server containers.Endpoint,
	backupType apiclient.PhysicalBackupType,
) apiclient.Database {
	t.Helper()

	var settings apiclient.DatabaseSettings
	require.NoError(t, settings.FromPostgresqlPhysicalSettings(newPhysicalDatabaseSettings(server, backupType)))

	return apptest.CreateDatabase(t, tm.owner, tm.workspace.Id, settings)
}

func newPhysicalDatabaseSettings(
	server containers.Endpoint,
	backupType apiclient.PhysicalBackupType,
) apiclient.PostgresqlPhysicalSettings {
	return apiclient.PostgresqlPhysicalSettings{
		Type:          apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host:          server.Host,
		Port:          server.Port,
		Username:      containers.PostgresUsername,
		Password:      new(containers.PostgresPassword),
		SslMode:       apiclient.PostgresqlSslModeDisable,
		SslClientCert: nil,
		SslClientKey:  nil,
		SslRootCert:   nil,
		SshTunnel: apiclient.SshTunnelSettings{
			IsEnabled:            false,
			AuthType:             nil,
			Host:                 nil,
			Port:                 nil,
			Username:             nil,
			Password:             nil,
			PrivateKey:           nil,
			PrivateKeyPassphrase: nil,
		},
		BackupType:          backupType,
		SystemIdentifier:    nil,
		Version:             nil,
		WalSegmentSizeBytes: nil,
	}
}

func getBackupConfig(t *testing.T, session apptest.Session, databaseID uuid.UUID) *apiclient.GetBackupConfigResponse {
	t.Helper()

	response, err := session.Client.GetBackupConfigWithResponse(t.Context(), databaseID)
	require.NoError(t, err)

	return response
}

func updateBackupConfig(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	input apiclient.BackupConfigInput,
) *apiclient.UpdateBackupConfigResponse {
	t.Helper()

	response, err := session.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID, input)
	require.NoError(t, err)

	return response
}

func listBackupConfigs(
	t *testing.T,
	session apptest.Session,
	params apiclient.ListBackupConfigsParams,
) *apiclient.ListBackupConfigsResponse {
	t.Helper()

	response, err := session.Client.ListBackupConfigsWithResponse(t.Context(), &params)
	require.NoError(t, err)

	return response
}

func listWorkspaceConfigDatabaseIDs(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []uuid.UUID {
	t.Helper()

	response := listBackupConfigs(t, session, apiclient.ListBackupConfigsParams{
		WorkspaceId: &workspaceID,
		StorageId:   nil,
		Limit:       nil,
		After:       nil,
	})
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	databaseIDs := make([]uuid.UUID, 0, len(response.JSON200.Items))
	for _, listedConfig := range response.JSON200.Items {
		databaseIDs = append(databaseIDs, listedConfig.DatabaseId)
	}

	return databaseIDs
}

// newLogicalBackupSettings is a valid logical config that differs from the logical default in every
// setting, so a test can tell a saved config from the default.
func newLogicalBackupSettings() apiclient.LogicalBackupSettings {
	timeOfDay := "02:30"
	weekday := 3

	return apiclient.LogicalBackupSettings{
		Type: apiclient.LogicalBackupSettingsTypeLOGICAL,
		Schedule: apiclient.Schedule{
			Type:           apiclient.ScheduleTypeWEEKLY,
			TimeOfDay:      &timeOfDay,
			Weekday:        &weekday,
			DayOfMonth:     nil,
			CronExpression: nil,
		},
		Retention: apiclient.LogicalRetention{
			Policy:    apiclient.LogicalRetentionPolicyCOUNT,
			Period:    apiclient.RetentionPeriodWEEK,
			Count:     5,
			GfsHours:  0,
			GfsDays:   0,
			GfsWeeks:  0,
			GfsMonths: 0,
			GfsYears:  0,
		},
		NotificationTriggers: []apiclient.LogicalNotificationTrigger{
			apiclient.LogicalNotificationTriggerBACKUPFAILED,
		},
		IsRetryIfFailed:     false,
		MaxFailedTriesCount: 2,
	}
}

func fromLogicalBackupSettings(t *testing.T, logical apiclient.LogicalBackupSettings) apiclient.BackupSettings {
	t.Helper()

	var settings apiclient.BackupSettings
	require.NoError(t, settings.FromLogicalBackupSettings(logical))

	return settings
}

func fromPhysicalBackupSettings(t *testing.T, physical apiclient.PhysicalBackupSettings) apiclient.BackupSettings {
	t.Helper()

	var settings apiclient.BackupSettings
	require.NoError(t, settings.FromPhysicalBackupSettings(physical))

	return settings
}

func newLogicalBackupConfigInput(
	t *testing.T,
	storageID *uuid.UUID,
	logical apiclient.LogicalBackupSettings,
) apiclient.BackupConfigInput {
	t.Helper()

	return apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        storageID,
		Encryption:       apiclient.BackupEncryptionNONE,
		Settings:         fromLogicalBackupSettings(t, logical),
	}
}

// saveLogicalBackupConfig saves newLogicalBackupSettings with the storage and fails the test unless the
// save succeeds.
func saveLogicalBackupConfig(
	t *testing.T,
	session apptest.Session,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) apiclient.BackupConfig {
	t.Helper()

	response := updateBackupConfig(t, session, databaseID,
		newLogicalBackupConfigInput(t, &storageID, newLogicalBackupSettings()))
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	return *response.JSON200
}

func getLogicalSettings(t *testing.T, backupConfig apiclient.BackupConfig) apiclient.LogicalBackupSettings {
	t.Helper()

	logical, err := backupConfig.Settings.AsLogicalBackupSettings()
	require.NoError(t, err)

	return logical
}

func getPhysicalSettings(t *testing.T, backupConfig apiclient.BackupConfig) apiclient.PhysicalBackupSettings {
	t.Helper()

	physical, err := backupConfig.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)

	return physical
}

func getSettingsType(t *testing.T, backupConfig apiclient.BackupConfig) string {
	t.Helper()

	discriminator, err := backupConfig.Settings.Discriminator()
	require.NoError(t, err)

	return discriminator
}

func countRows(t *testing.T, tm team, query string, values ...any) int {
	t.Helper()

	var count int
	tm.app.QuerySQL(t, &count, query, values...)

	return count
}

func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.Equal(t, expectedCode, problem.Code, string(body))
}

func requireInvalidField(t *testing.T, status int, body []byte, expectedField string) {
	t.Helper()

	requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.NotNil(t, problem.Errors, string(body))

	fields := make([]string, 0, len(*problem.Errors))
	for _, fieldError := range *problem.Errors {
		fields = append(fields, fieldError.Field)
	}

	require.Contains(t, fields, expectedField, string(body))
}
