package upgrade

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type retirementSnapshotSpec struct {
	resources upgradeResources
	image     string
}

func getStoppedRetirementSnapshot(t *testing.T, docker *dockerHost, spec retirementSnapshotSpec) string {
	t.Helper()

	require.NoError(t, docker.StartContainer(t.Context(), containers.RunSpec{
		Name:  spec.resources.sourceContainerName,
		Image: spec.image,
		Entrypoint: []string{"sh", "-c", `printf 'local all all trust\n' > /tmp/retirement-pg_hba
exec gosu databasus /usr/lib/postgresql/17/bin/postgres -D /databasus-data/pgdata -c hba_file=/tmp/retirement-pg_hba -c unix_socket_directories=/tmp -c listen_addresses=''`},
		Env:           nil,
		Mount:         spec.resources.volumeName + ":" + instanceDataPath,
		NetworkName:   spec.resources.networkName,
		NetworkAlias:  spec.resources.sourceContainerName,
		PublishedPort: "5437/tcp",
	}))

	testwait.Until(t, "the snapshot database to answer read-only queries", func(ctx context.Context) error {
		_, err := docker.Execute(ctx, spec.resources.sourceContainerName,
			[]string{"psql", "-h", "/tmp", "-p", "5437", "-U", "postgres", "-d", "databasus", "-AtX", "-c", "SELECT 1"})
		return err
	})

	snapshot, err := docker.Execute(t.Context(), spec.resources.sourceContainerName,
		[]string{"sh", "-c", `psql -h /tmp -p 5437 -U postgres -d databasus -AtXq -v ON_ERROR_STOP=1 <<'SQL'
BEGIN READ ONLY;
SELECT format('SELECT %L || coalesce(jsonb_agg(to_jsonb(records) ORDER BY to_jsonb(records)::text), ''[]'')::text FROM %I records', tablename, tablename)
FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename
\gexec
ROLLBACK;
SQL`})
	require.NoError(t, err)
	require.NotEmpty(t, snapshot)

	require.NoError(t, docker.StopContainer(t.Context(), spec.resources.sourceContainerName))
	removeRetirementContainer(t, docker, spec.resources.sourceContainerName)

	return snapshot
}

func getSupportedRetirementSecrets(t *testing.T, docker *dockerHost, containerName string) string {
	t.Helper()

	secrets, err := docker.Execute(t.Context(), containerName,
		[]string{"sh", "-c", `psql "$(cat /dev/shm/databasus-database-dsn)" -AtXq -v ON_ERROR_STOP=1 <<'SQL'
BEGIN READ ONLY;
SELECT format('SELECT jsonb_agg(jsonb_build_array(storage_id, s3_access_key, s3_secret_key) ORDER BY storage_id)::text FROM %I', tablename)
FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('s3_storages', 'storages_s3')
\gexec
ROLLBACK;
SQL`})
	require.NoError(t, err)
	require.NotEmpty(t, secrets)

	return secrets
}

func removeRetirementContainer(t *testing.T, docker *dockerHost, containerName string) {
	t.Helper()

	_, err := docker.client.ContainerRemove(t.Context(), containerName, client.ContainerRemoveOptions{Force: true})
	require.NoError(t, err)
}
