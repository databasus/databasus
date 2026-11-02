package healthcheck_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestHealthcheckPhysicalProbe(t *testing.T) {
	t.Parallel()

	t.Run("records success and failure for a physical PostgreSQL database", func(t *testing.T) {
		t.Parallel()

		team := enginetest.NewTeam(t)
		endpoint := containers.StartPhysicalPostgres(t, "postgres:17")
		userDatabase := team.CreateDatabase(t, apiclient.PostgresqlPhysicalSettings{
			Type:       apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
			Host:       endpoint.Host,
			Port:       endpoint.Port,
			Username:   containers.PostgresUsername,
			Password:   new(containers.PostgresPassword),
			SslMode:    apiclient.PostgresqlSslModeDisable,
			SshTunnel:  enginetest.NewDisabledSSHTunnelSettings(),
			BackupType: apiclient.PhysicalBackupTypeFULL,
		})

		healthchecks := healthcheckTeam{Team: team, Database: userDatabase}
		healthchecks.pass(t)
		attempts := healthchecks.getAttempts(t)
		require.Len(t, attempts, 1)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), attempts[0].Status)

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET port = ? WHERE database_id = ?",
			closedPort(t),
			userDatabase.Id,
		)

		team.App.Clock.Advance(time.Minute)
		healthchecks.pass(t)
		attempts = healthchecks.getAttempts(t)
		require.Len(t, attempts, 2)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), attempts[0].Status)
	})
}
