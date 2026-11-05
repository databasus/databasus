package healthcheck_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type healthcheckTeam struct {
	enginetest.Team
	Database apiclient.Database
}

func newHealthcheckTeam(t *testing.T) healthcheckTeam {
	t.Helper()
	team := enginetest.NewTeam(t)
	endpoint := containers.StartPostgres(t, "postgres:18")
	database := team.CreateDatabase(
		t,
		apiclient.PostgresqlLogicalSettings{
			Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
			Host:               endpoint.Host,
			Port:               endpoint.Port,
			Username:           containers.PostgresUsername,
			Password:           new(containers.PostgresPassword),
			Database:           new(containers.PostgresDatabase),
			SslMode:            apiclient.PostgresqlSslModeDisable,
			SshTunnel:          enginetest.NewDisabledSSHTunnelSettings(),
			CpuCount:           1,
			SslClientCert:      nil,
			SslClientKey:       nil,
			SslRootCert:        nil,
			IncludeSchemas:     nil,
			ExcludeTables:      nil,
			ExcludeTableData:   nil,
			IsSkipUserMappings: false,
			Version:            nil,
		},
	)
	return healthcheckTeam{Team: team, Database: database}
}

func (team healthcheckTeam) getConfig(t *testing.T) apiclient.HealthcheckConfig {
	t.Helper()
	response, err := team.Owner.Client.GetHealthcheckConfigWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return *response.JSON200
}

func configInput(config apiclient.HealthcheckConfig) apiclient.HealthcheckConfigInput {
	return apiclient.HealthcheckConfigInput{
		IsHealthcheckEnabled:            config.IsHealthcheckEnabled,
		IsNotificationEnabled:           config.IsNotificationEnabled,
		IntervalMinutes:                 config.IntervalMinutes,
		FailedAttemptsBeforeUnavailable: config.FailedAttemptsBeforeUnavailable,
		AttemptRetentionDays:            config.AttemptRetentionDays,
	}
}

func (team healthcheckTeam) saveConfig(
	t *testing.T,
	config apiclient.HealthcheckConfigInput,
) apiclient.HealthcheckConfig {
	t.Helper()
	response, err := team.Owner.Client.UpdateHealthcheckConfigWithResponse(t.Context(), team.Database.Id, config)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return *response.JSON200
}

func (team healthcheckTeam) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()
	member := team.App.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, team.Owner, team.Workspace.Id, member.Email, role)
	return member
}

func (team healthcheckTeam) getAttempts(t *testing.T) []apiclient.HealthcheckAttempt {
	t.Helper()
	response, err := team.Owner.Client.ListHealthcheckAttemptsWithResponse(
		t.Context(),
		&apiclient.ListHealthcheckAttemptsParams{
			DatabaseId: team.Database.Id,
			Since:      time.Unix(0, 0).UTC(),
			Limit:      nil,
			After:      nil,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return response.JSON200.Items
}

func (team healthcheckTeam) getHealthStatus(t *testing.T) *apiclient.DatabaseHealthStatus {
	t.Helper()
	response, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return response.JSON200.HealthStatus
}

func (team healthcheckTeam) pass(t *testing.T) {
	t.Helper()
	team.App.RunJobPass(t, "healthcheck_probe")
	team.App.WaitForNotificationDeliveries(t)
}

func closedPort(t *testing.T) int {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, isTCP := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCP)
	port := address.Port
	require.NoError(t, listener.Close())
	return port
}

func (team healthcheckTeam) makeUnreachable(t *testing.T) {
	t.Helper()
	team.App.ExecuteSQL(
		t,
		"UPDATE databases_postgresql_logical SET port = ? WHERE database_id = ?",
		closedPort(t),
		team.Database.Id,
	)
}
