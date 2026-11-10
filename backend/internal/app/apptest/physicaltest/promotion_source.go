package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type promotionSourceSpec struct {
	Primary    containers.Endpoint
	Standby    containers.Endpoint
	BackupType apiclient.PhysicalBackupType
}

func (team *Team) createPromotionDatabase(t *testing.T, spec promotionSourceSpec) {
	t.Helper()

	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint: spec.Primary, BackupType: spec.BackupType,
		Tunnel: enginetest.NewDisabledSSHTunnelSettings(),
	})
	team.useReplicationOnlyUser(t)
	primary := openConnAt(t, spec.Primary.Host, spec.Primary.Port)
	_, err := primary.Exec(t.Context(), "CREATE TABLE backup_permission_probe(id integer)")
	require.NoError(t, err)
	standby := openConnAt(t, spec.Standby.Host, spec.Standby.Port)
	testwait.Until(t, "the backup role is replayed on standby", func(ctx context.Context) error {
		var hasRole bool
		if err := standby.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1) AND to_regclass('backup_permission_probe') IS NOT NULL", team.Settings.Username).
			Scan(&hasRole); err != nil {
			return err
		}
		if !hasRole {
			return fmt.Errorf("backup role has not replayed")
		}

		return nil
	})

	team.Settings.Host = spec.Standby.Host
	team.Settings.Port = spec.Standby.Port
	team.Database = team.UpdateDatabaseSettings(t, team.Database, team.Settings)
	team.requireRestrictedPromotionRole(t)
}

func (team Team) requireRestrictedPromotionRole(t *testing.T) {
	t.Helper()

	connectionConfig, err := pgx.ParseConfig(
		fmt.Sprintf("host=%s port=%d dbname=testdb sslmode=disable", team.Settings.Host, team.Settings.Port),
	)
	require.NoError(t, err)
	connectionConfig.User = team.Settings.Username
	connectionConfig.Password = *team.Settings.Password
	connection, err := pgx.ConnectConfig(t.Context(), connectionConfig)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	for _, role := range []string{"pg_read_server_files", "pg_read_all_data", "pg_write_server_files", "pg_execute_server_program"} {
		var isMember bool
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT pg_has_role(current_user,$1,'MEMBER')", role).Scan(&isMember),
		)
		require.False(t, isMember, role)
	}

	var canReadFiles bool
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT has_function_privilege(current_user,'pg_read_binary_file(text)','EXECUTE')").
			Scan(&canReadFiles),
	)
	require.False(t, canReadFiles)

	_, err = connection.Exec(t.Context(), "SELECT * FROM backup_permission_probe")
	var permissionError *pgconn.PgError
	require.ErrorAs(t, err, &permissionError)
	require.Equal(t, "42501", permissionError.Code)
}
