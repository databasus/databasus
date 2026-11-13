package physicaltest

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type Team struct {
	enginetest.Team
	Database apiclient.Database
	Settings apiclient.PostgresqlPhysicalSettings
}

func newTeam(t *testing.T, options ...apptest.Option) Team {
	t.Helper()

	return Team{Team: enginetest.NewTeam(t, append([]apptest.Option{apptest.WithPhysicalBackupLoops()}, options...)...)}
}

type physicalDatabaseSpec struct {
	Endpoint   containers.Endpoint
	BackupType apiclient.PhysicalBackupType
	Tunnel     apiclient.SshTunnelSettings
	Access     sourceAccess
}

// sourceAccess is how Databasus logs in to a source. The zero value is the administrator over plain
// TCP, which then lets Databasus create its limited user; a source with a declared role and TLS
// supplies both instead.
type sourceAccess struct {
	Username        string
	Password        string
	SslMode         apiclient.PostgresqlSslMode
	RootCertificate *string
}

func (team *Team) createDatabase(t *testing.T, spec physicalDatabaseSpec) {
	t.Helper()

	access := spec.Access
	if access.Username == "" {
		access = sourceAccess{
			Username: restoredPgUser,
			Password: restoredPgPassword,
			SslMode:  apiclient.PostgresqlSslModeDisable,
		}
	}

	team.Settings = apiclient.PostgresqlPhysicalSettings{
		Type:        apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host:        spec.Endpoint.Host,
		Port:        spec.Endpoint.Port,
		Username:    access.Username,
		Password:    &access.Password,
		SslMode:     access.SslMode,
		SslRootCert: access.RootCertificate,
		SshTunnel:   spec.Tunnel,
		BackupType:  spec.BackupType,
	}
	team.Database = team.CreateDatabase(t, team.Settings)
}

type physicalSourceSpec struct {
	Image      string
	BackupType apiclient.PhysicalBackupType
	Options    []containers.PhysicalPostgresOption
}

func setupReplicationOnlyTeam(t *testing.T, spec physicalSourceSpec) Team {
	t.Helper()

	team := newTeam(t)
	source := containers.StartPhysicalPostgres(t, spec.Image, spec.Options...)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   source,
			BackupType: spec.BackupType,
			Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		},
	)
	team.useReplicationOnlyUser(t)
	return team
}

func (team *Team) useReplicationOnlyUser(t *testing.T) {
	t.Helper()

	limitedUser := team.CreateLimitedUser(t, team.Database)
	require.NotEmpty(t, limitedUser.Username)
	require.NotEmpty(t, limitedUser.Password)
	team.Settings.Username = limitedUser.Username
	team.Settings.Password = &limitedUser.Password
	team.Database = team.UpdateDatabaseSettings(t, team.Database, team.Settings)

	team.requireRestrictedRoleAttributes(t, openConnAt(t, team.Settings.Host, team.Settings.Port))
}

// requireRestrictedRoleAttributes checks the role Databasus uses, whoever created it.
func (team *Team) requireRestrictedRoleAttributes(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	var isReplication, isSuperuser, canCreateRole, canCreateDatabase, canBypassRLS bool
	require.NoError(
		t,
		conn.QueryRow(t.Context(), `SELECT rolreplication, rolsuper, rolcreaterole, rolcreatedb, rolbypassrls FROM pg_roles WHERE rolname = $1`, team.Settings.Username).
			Scan(&isReplication, &isSuperuser, &canCreateRole, &canCreateDatabase, &canBypassRLS),
	)

	require.True(t, isReplication)
	require.False(t, isSuperuser)
	require.False(t, canCreateRole)
	require.False(t, canCreateDatabase)
	require.False(t, canBypassRLS)
}

func cronSchedule() apiclient.Schedule {
	expression := "0 0 1 1 *"
	return apiclient.Schedule{Type: apiclient.ScheduleTypeCRON, CronExpression: &expression}
}

func (team Team) enableBackups(t *testing.T, isWalStream bool) {
	t.Helper()

	incrementalSchedule := cronSchedule()
	physicalSettings := apiclient.PhysicalBackupSettings{
		Type:                      apiclient.PhysicalBackupSettingsTypePHYSICAL,
		FullBackupSchedule:        cronSchedule(),
		IncrementalBackupSchedule: &incrementalSchedule,
		Retention: apiclient.PhysicalRetention{
			Policy:      apiclient.PhysicalRetentionPolicyCHAINS,
			ChainsCount: 50,
		},
		NotificationTriggers: []apiclient.NotificationTrigger{},
	}

	if isWalStream {
		physicalSettings.WalLagThresholdBytes = 64 * 1024 * 1024
	}

	if team.Settings.BackupType == apiclient.PhysicalBackupTypeFULL {
		physicalSettings.IncrementalBackupSchedule = nil
		physicalSettings.Retention = apiclient.PhysicalRetention{
			Policy:      apiclient.PhysicalRetentionPolicyFULLBACKUPS,
			FullBackups: &apiclient.FullBackupsRetention{Policy: apiclient.FullBackupsRetentionPolicyLASTN, Count: 50},
		}
	}

	var settings apiclient.BackupSettings
	require.NoError(t, settings.FromPhysicalBackupSettings(physicalSettings))

	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &team.StorageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

type physicalResponseSpec struct {
	Call        apptest.PhysicalRequest
	Status      int
	Destination any
}

func (team Team) call(t *testing.T, spec physicalResponseSpec) []byte {
	t.Helper()

	responseStatus, body := team.App.CallPhysicalAPI(t, team.Owner, spec.Call)
	require.Equal(t, spec.Status, responseStatus, "%s", body)
	if spec.Destination != nil {
		require.NoError(t, json.Unmarshal(body, spec.Destination))
	}
	return body
}

func (team Team) triggerIncremental(t *testing.T) {
	t.Helper()

	team.call(
		t,
		physicalResponseSpec{
			Call:        apptest.PhysicalStartRequest(team.Database.Id, "INCREMENTAL"),
			Status:      http.StatusAccepted,
			Destination: nil,
		},
	)
}

func (team Team) deleteDatabase(t *testing.T) {
	t.Helper()

	response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
}
