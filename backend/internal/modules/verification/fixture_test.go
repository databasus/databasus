package verification_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type verificationTeam struct {
	enginetest.Team
	Database apiclient.Database
}

func newVerificationTeam(t *testing.T) verificationTeam {
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
	return verificationTeam{Team: team, Database: database}
}

func (team verificationTeam) backup(t *testing.T) apiclient.LogicalBackup {
	t.Helper()
	return team.BackUp(t, team.Database.Id, apiclient.BackupEncryptionNONE)
}

func (team verificationTeam) start(t *testing.T, backupID uuid.UUID) apiclient.Verification {
	t.Helper()
	response, err := team.Owner.Client.StartVerificationWithResponse(
		t.Context(),
		apiclient.StartVerificationInput{BackupId: backupID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
	return *response.JSON202
}

func (team verificationTeam) read(t *testing.T, id uuid.UUID) apiclient.VerificationDetails {
	t.Helper()
	response, err := team.Owner.Client.GetVerificationWithResponse(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	return *response.JSON200
}

func (team verificationTeam) list(t *testing.T) []apiclient.Verification {
	t.Helper()
	response, err := team.Owner.Client.ListVerificationsWithResponse(
		t.Context(),
		&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id, Limit: new(100), After: nil},
	)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return response.JSON200.Items
}

func (team verificationTeam) config(t *testing.T) apiclient.VerificationConfig {
	t.Helper()
	response, err := team.Owner.Client.GetVerificationConfigWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return *response.JSON200
}

func configInput(config apiclient.VerificationConfig) apiclient.VerificationConfigInput {
	return apiclient.VerificationConfigInput{
		IsAutomaticVerificationEnabled: config.IsAutomaticVerificationEnabled,
		ScheduleType:                   config.ScheduleType,
		Schedule:                       config.Schedule,
		NotificationTriggers:           config.NotificationTriggers,
	}
}

func (team verificationTeam) saveConfig(
	t *testing.T,
	config apiclient.VerificationConfigInput,
) apiclient.VerificationConfig {
	t.Helper()
	response, err := team.Owner.Client.UpdateVerificationConfigWithResponse(t.Context(), team.Database.Id, config)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))
	return *response.JSON200
}

func (team verificationTeam) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()
	member := team.App.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, team.Owner, team.Workspace.Id, member.Email, role)
	return member
}
