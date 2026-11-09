package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type retirementSnapshotSpec struct {
	resources upgradeResources
	image     string
}

func getStoppedRetirementSnapshot(t *testing.T, docker *dockerHost, spec retirementSnapshotSpec) string {
	t.Helper()

	require.NoError(t, docker.startContainer(t.Context(), containerSpec{
		name:  spec.resources.sourceContainerName,
		image: spec.image,
		entrypoint: []string{"sh", "-c", `printf 'local all all trust\n' > /tmp/retirement-pg_hba
exec gosu databasus /usr/lib/postgresql/17/bin/postgres -D /databasus-data/pgdata -c hba_file=/tmp/retirement-pg_hba -c unix_socket_directories=/tmp -c listen_addresses=''`},
		env:           nil,
		volumeMount:   spec.resources.volumeName + ":" + instanceDataPath,
		networkName:   spec.resources.networkName,
		networkAlias:  spec.resources.sourceContainerName,
		publishedPort: "5437/tcp",
	}))

	testwait.Until(t, "the snapshot database to answer read-only queries", func(ctx context.Context) error {
		_, err := docker.executeRetirementCommand(ctx, spec.resources.sourceContainerName,
			[]string{"psql", "-h", "/tmp", "-p", "5437", "-U", "postgres", "-d", "databasus", "-AtX", "-c", "SELECT 1"})
		return err
	})

	snapshot, err := docker.executeRetirementCommand(t.Context(), spec.resources.sourceContainerName,
		[]string{"sh", "-c", `psql -h /tmp -p 5437 -U postgres -d databasus -AtXq -v ON_ERROR_STOP=1 <<'SQL'
BEGIN READ ONLY;
SELECT format('SELECT %L || coalesce(jsonb_agg(to_jsonb(records) ORDER BY to_jsonb(records)::text), ''[]'')::text FROM %I records', tablename, tablename)
FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename
\gexec
ROLLBACK;
SQL`})
	require.NoError(t, err)
	require.NotEmpty(t, snapshot)

	require.NoError(t, docker.stopContainer(t.Context(), spec.resources.sourceContainerName))
	removeRetirementContainer(t, docker, spec.resources.sourceContainerName)

	return snapshot
}

func getSupportedRetirementSecrets(t *testing.T, docker *dockerHost, containerName string) string {
	t.Helper()

	secrets, err := docker.executeRetirementCommand(t.Context(), containerName,
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

func (d *dockerHost) executeRetirementCommand(
	ctx context.Context,
	containerName string,
	command []string,
) (string, error) {
	created, err := d.client.ExecCreate(ctx, containerName, client.ExecCreateOptions{
		Cmd: command, AttachStdout: true, AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("create retirement inspection: %w", err)
	}

	attached, err := d.client.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attach retirement inspection: %w", err)
	}
	defer attached.Close()

	var output bytes.Buffer
	_, err = stdcopy.StdCopy(&output, io.Discard, attached.Reader)
	if err != nil {
		return "", fmt.Errorf("read retirement inspection: %w", err)
	}

	inspection, err := d.client.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspect retirement command: %w", err)
	}

	if inspection.ExitCode != 0 {
		return "", fmt.Errorf("retirement inspection exited with code %d", inspection.ExitCode)
	}

	return output.String(), nil
}

func removeRetirementContainer(t *testing.T, docker *dockerHost, containerName string) {
	t.Helper()

	_, err := docker.client.ContainerRemove(t.Context(), containerName, client.ContainerRemoveOptions{Force: true})
	require.NoError(t, err)
}
