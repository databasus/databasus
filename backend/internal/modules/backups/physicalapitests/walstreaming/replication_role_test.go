package walstreaming_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func (team *walTeam) useReplicationRole(t *testing.T) string {
	t.Helper()
	roleName := "backup_" + uuid.NewString()[:8]
	source := team.openSource(t)
	_, err := source.Exec(
		t.Context(),
		fmt.Sprintf(
			"CREATE ROLE %s LOGIN REPLICATION PASSWORD 'replication-test-password'",
			pgx.Identifier{roleName}.Sanitize(),
		),
	)
	require.NoError(t, err)
	_, err = source.Exec(t.Context(), "GRANT pg_monitor TO "+pgx.Identifier{roleName}.Sanitize())
	require.NoError(t, err)
	_, err = source.Exec(
		t.Context(),
		"GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "+pgx.Identifier{roleName}.Sanitize(),
	)
	require.NoError(t, err)

	settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
	require.NoError(t, err)
	settings.Username = roleName
	settings.Password = new("replication-test-password")
	var databaseSettings apiclient.DatabaseSettings
	require.NoError(t, databaseSettings.FromPostgresqlPhysicalSettings(settings))

	response, err := team.Owner.Client.UpdateDatabaseWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.DatabaseUpdate{
			Name:        team.Database.Name,
			NotifierIds: team.Database.NotifierIds,
			Settings:    databaseSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	team.Database = *response.JSON200
	return roleName
}
