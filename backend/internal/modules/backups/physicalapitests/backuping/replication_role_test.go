package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func (team *backupTeam) useReplicationRole(t *testing.T) string {
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

func (team *backupTeam) denyPhysicalReplication(t *testing.T) {
	t.Helper()
	source := team.openSource(t)
	var hbaPath string
	require.NoError(t, source.QueryRow(t.Context(), "SHOW hba_file").Scan(&hbaPath))
	var previousLoad time.Time
	require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_conf_load_time()").Scan(&previousLoad))

	_, err := source.Exec(
		t.Context(),
		"COPY (SELECT hba_line FROM (VALUES ('local all all trust'),('host all all all scram-sha-256')) AS lines(hba_line)) TO '"+strings.ReplaceAll(
			hbaPath,
			"'",
			"''",
		)+"'",
	)
	require.NoError(t, err)

	_, err = source.Exec(t.Context(), "SELECT pg_reload_conf()")
	require.NoError(t, err)

	testwait.Until(t, "PostgreSQL reloads the physical replication rejection", func(context.Context) error {
		var loadedAt time.Time
		if err := source.QueryRow(t.Context(), "SELECT pg_conf_load_time()").Scan(&loadedAt); err != nil {
			return err
		}
		if !loadedAt.After(previousLoad) {
			return fmt.Errorf("the source has not reloaded its HBA file")
		}
		return nil
	})
}
