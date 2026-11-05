package e2e_test

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func getRestoreImages() []string {
	return []string{
		"databasus/verification-postgres:12",
		"databasus/verification-postgres:13",
		"databasus/verification-postgres:14",
		"databasus/verification-postgres:15",
		"databasus/verification-postgres:16",
		"databasus/verification-postgres:17",
		"databasus/verification-postgres:18",
	}
}

const timescaleImage = "timescale/timescaledb:2.17.0-pg17"

const ordinaryTables = `CREATE TABLE public.t_a (id int PRIMARY KEY, name text);
CREATE TABLE public.t_b (id int PRIMARY KEY, a_id int NOT NULL REFERENCES public.t_a(id), value text);
INSERT INTO public.t_a VALUES (1,'alpha'),(2,'beta'),(3,'gamma');
INSERT INTO public.t_b VALUES (1,1,'x'),(2,1,'y'),(3,2,'z'),(4,3,'w'),(5,3,'v');`

func seedSource(t *testing.T, endpoint containers.Endpoint, isTimescale bool) apiclient.PostgresqlLogicalSettings {
	t.Helper()
	connect := func(database string) *pgx.Conn {
		connection, err := pgx.Connect(
			t.Context(),
			fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				endpoint.Host,
				endpoint.Port,
				containers.PostgresUsername,
				containers.PostgresPassword,
				database,
			),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = connection.Close(t.Context()) })
		return connection
	}

	admin := connect(containers.PostgresDatabase)
	databaseName := "agent_" + enginetest.NewUniqueSuffix()
	_, err := admin.Exec(t.Context(), "CREATE DATABASE "+databaseName)
	require.NoError(t, err)
	source := connect(databaseName)

	statements := ordinaryTables
	if isTimescale {
		role := "ts_app_" + enginetest.NewUniqueSuffix()
		statements = fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE ROLE %[1]s LOGIN;
GRANT CREATE ON SCHEMA public TO %[1]s;
SET ROLE %[1]s;
CREATE TABLE public.sensor_data (time TIMESTAMPTZ NOT NULL,sensor_id int NOT NULL,temperature double precision NOT NULL);
SELECT create_hypertable('public.sensor_data','time');
SELECT add_retention_policy('public.sensor_data',INTERVAL '1000 years');
INSERT INTO public.sensor_data SELECT ts,1,23.5 FROM generate_series('2024-01-01'::timestamptz,'2024-03-01'::timestamptz,interval '1 hour') AS ts;
RESET ROLE;`, role)
	}

	_, err = source.Exec(t.Context(), statements)
	require.NoError(t, err)
	return apiclient.PostgresqlLogicalSettings{
		Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:               endpoint.Host,
		Port:               endpoint.Port,
		Username:           containers.PostgresUsername,
		Password:           new(containers.PostgresPassword),
		Database:           &databaseName,
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
	}
}
