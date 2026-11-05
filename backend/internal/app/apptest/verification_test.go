package apptest_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestVerificationHelpers(t *testing.T) {
	t.Parallel()

	t.Run("reads a pending manual verification of a completed backup", func(t *testing.T) {
		t.Parallel()

		team, backup := newVerificationBackup(t)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		require.NotEmpty(t, agent.Token)
		agents := team.App.ListVerificationAgents(t, team.Admin)
		require.Len(t, agents, 1)
		assert.Equal(t, agent.Agent.ID, agents[0].ID)

		verificationID := team.App.EnqueueVerification(t, team.Owner, backup.Id)
		verification := team.App.GetVerification(t, team.Owner, verificationID)
		assert.Equal(t, verificationID, verification.ID)
		assert.Equal(t, "PENDING", verification.Status)
		assert.Equal(t, 1, verification.AttemptCount)

		read, err := team.Owner.Client.GetVerificationWithResponse(t.Context(), verificationID)
		require.NoError(t, err)
		require.NotNil(t, read.JSON200)
		assert.Equal(t, verification.ID, read.JSON200.Id)
		assert.Equal(t, verification.Status, string(read.JSON200.Status))
		assert.Empty(t, read.JSON200.TableStats)
	})
	t.Run("detects a truncated download while the backup stays completed", func(t *testing.T) {
		t.Parallel()

		team, backup := newVerificationBackup(t)
		damagedBytes := []byte("damaged backup bytes")
		team.App.DamageLocalBackup(t, backup.Id, damagedBytes)
		token, err := team.Owner.Client.CreateDownloadTokenWithResponse(
			t.Context(),
			apiclient.DownloadTokenCreate{BackupId: backup.Id},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, token.StatusCode(), string(token.Body))

		response, err := team.Owner.Client.DownloadLogicalBackup(
			t.Context(),
			backup.Id,
			&apiclient.DownloadLogicalBackupParams{Token: token.JSON201.Token},
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()
		contents, err := io.ReadAll(response.Body)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, damagedBytes, contents)
		assert.Equal(t, int64(backup.SizeMb*1024*1024), response.ContentLength)

		persistedBackup, err := team.Owner.Client.GetLogicalBackupWithResponse(t.Context(), backup.Id)
		require.NoError(t, err)
		require.NotNil(t, persistedBackup.JSON200)
		assert.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, persistedBackup.JSON200.Status)
	})
}

func newVerificationBackup(t *testing.T) (enginetest.Team, apiclient.LogicalBackup) {
	t.Helper()
	endpoint := containers.StartPostgres(t, "postgres:18")
	team := enginetest.NewTeam(t)
	source, err := pgx.Connect(
		t.Context(),
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			endpoint.Host,
			endpoint.Port,
			containers.PostgresUsername,
			containers.PostgresPassword,
			containers.PostgresDatabase,
		),
	)
	require.NoError(t, err)
	defer func() { _ = source.Close(t.Context()) }()
	_, err = source.Exec(t.Context(), "CREATE TABLE payload (id int); INSERT INTO payload VALUES (1)")
	require.NoError(t, err)
	database := team.CreateDatabase(t, apiclient.PostgresqlLogicalSettings{
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
	})
	return team, team.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
}
