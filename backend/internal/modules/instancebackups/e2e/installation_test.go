package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	dataDirectory          = "/databasus-data"
	instancePublishedPort  = "4005/tcp"
	recoveryArchivePath    = dataDirectory + "/recovery/backup.dbbackup"
	recoveryKeyPath        = dataDirectory + "/recovery/master.key"
	installedKeyPath       = dataDirectory + "/secret.key"
	runtimeIdentityDefault = 999
)

var errInstanceExited = errors.New("the instance exited")

// dataLayout is how a deployment stores /databasus-data: the two layouts the documentation names.
type dataLayout string

const (
	namedVolumeLayout dataLayout = "named volume"
	bindMountLayout   dataLayout = "bind mount"
)

// installation is one Databasus deployment with its own data. Every start runs a new container on
// that data, the way an operator recreates a container with changed settings.
type installation struct {
	cycle         *recoveryCycle
	name          string
	mount         string
	hostDirectory string
	owner         containers.FileOwner
	startCount    int
	resources     containers.DockerResources
}

func (c *recoveryCycle) newInstallation(t *testing.T, label string, layout dataLayout) *installation {
	t.Helper()

	created := &installation{
		cycle: c,
		name:  c.prefix + "-" + label,
		owner: getRuntimeOwner(),
	}
	t.Cleanup(func() { c.removeResources(t, created.resources) })

	switch layout {
	case namedVolumeLayout:
		volumeName := created.name + "-data"
		created.resources.VolumeNames = append(created.resources.VolumeNames, volumeName)
		require.NoError(t, c.docker.CreateVolume(t.Context(), volumeName))
		created.mount = volumeName + ":" + dataDirectory
	case bindMountLayout:
		created.hostDirectory = t.TempDir()
		created.mount = created.hostDirectory + ":" + dataDirectory
	default:
		t.Fatalf("unknown data layout %s", layout)
	}

	return created
}

// getRuntimeOwner is the identity Databasus runs as inside every container: the host user, so a
// bind-mounted directory stays the test's to read and remove.
func getRuntimeOwner() containers.FileOwner {
	uid, gid := os.Getuid(), os.Getgid()
	if uid == 0 || gid == 0 {
		return containers.FileOwner{UID: runtimeIdentityDefault, GID: runtimeIdentityDefault}
	}

	return containers.FileOwner{UID: uid, GID: gid}
}

// getRecoveryEnvironment configures both recovery paths at the files putRecoveryFiles writes.
func getRecoveryEnvironment() []string {
	return []string{"RECOVERY_DUMP_PATH=" + recoveryArchivePath, "RECOVERY_MASTER_KEY_PATH=" + recoveryKeyPath}
}

// putRecoveryFiles adds the archive and the key to the data, as an owner does before a restart.
func (i *installation) putRecoveryFiles(t *testing.T, archive, key []byte) {
	t.Helper()

	i.putFile(t, recoveryArchivePath, archive)
	i.putFile(t, recoveryKeyPath, key)
}

// putFile writes a file under /databasus-data while no container runs on the data.
func (i *installation) putFile(t *testing.T, containerPath string, content []byte) {
	t.Helper()

	if i.hostDirectory != "" {
		hostPath := i.getHostPath(t, containerPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(hostPath), 0o700))
		require.NoError(t, os.WriteFile(hostPath, content, 0o600))

		return
	}

	idleContainer := i.createIdleContainer(t)
	require.NoError(t, i.cycle.docker.CopyFileToContainer(t.Context(), idleContainer, containerPath, content, i.owner))
	require.NoError(t, i.cycle.docker.RemoveContainer(t.Context(), idleContainer))
}

// readFile reads a file under /databasus-data while no container runs on the data.
func (i *installation) readFile(t *testing.T, containerPath string) []byte {
	t.Helper()

	if i.hostDirectory != "" {
		content, err := os.ReadFile(i.getHostPath(t, containerPath))
		require.NoError(t, err)

		return content
	}

	idleContainer := i.createIdleContainer(t)
	content, err := i.cycle.docker.ReadFileFromContainer(t.Context(), idleContainer, containerPath)
	require.NoError(t, err)
	require.NoError(t, i.cycle.docker.RemoveContainer(t.Context(), idleContainer))

	return content
}

func (i *installation) getHostPath(t *testing.T, containerPath string) string {
	t.Helper()

	relativePath, err := filepath.Rel(dataDirectory, containerPath)
	require.NoError(t, err)

	return filepath.Join(i.hostDirectory, relativePath)
}

func (i *installation) createIdleContainer(t *testing.T) string {
	t.Helper()

	containerName := i.nextContainerName("files")
	require.NoError(t, i.cycle.docker.CreateContainer(t.Context(), containers.RunSpec{
		Name:          containerName,
		Image:         i.cycle.image,
		Entrypoint:    []string{"true"},
		Env:           nil,
		Mount:         i.mount,
		NetworkName:   i.cycle.networkName,
		NetworkAlias:  containerName,
		PublishedPort: "",
	}))

	return containerName
}

// start runs Databasus on the data and waits until it answers its health check.
func (i *installation) start(t *testing.T, environment ...string) runningInstance {
	t.Helper()

	started := i.startContainer(t, environment)
	started.waitForHealth(t)

	return started
}

// startUntilExit runs Databasus on the data and waits until it exits by itself, as it does when it
// refuses to start.
func (i *installation) startUntilExit(t *testing.T, environment ...string) exitedInstance {
	t.Helper()

	started := i.startContainer(t, environment)

	return started.waitForExit(t)
}

func (i *installation) startContainer(t *testing.T, environment []string) runningInstance {
	t.Helper()

	containerName := i.nextContainerName("instance")
	require.NoError(t, i.cycle.docker.StartContainer(t.Context(), containers.RunSpec{
		Name:  containerName,
		Image: i.cycle.image,
		Env: append([]string{
			"IS_DISABLE_ANONYMOUS_TELEMETRY=true",
			"PUID=" + strconv.Itoa(i.owner.UID),
			"PGID=" + strconv.Itoa(i.owner.GID),
		}, environment...),
		Entrypoint:    nil,
		Mount:         i.mount,
		NetworkName:   i.cycle.networkName,
		NetworkAlias:  containerName,
		PublishedPort: instancePublishedPort,
	}))

	return runningInstance{cycle: i.cycle, containerName: containerName}
}

func (i *installation) nextContainerName(role string) string {
	i.startCount++
	containerName := fmt.Sprintf("%s-%s-%d", i.name, role, i.startCount)
	i.resources.ContainerNames = append(i.resources.ContainerNames, containerName)

	return containerName
}

// runningInstance is one container of an installation.
type runningInstance struct {
	cycle         *recoveryCycle
	containerName string
}

type exitedInstance struct {
	exitCode int
	logs     string
}

func (r runningInstance) getURL(t *testing.T) string {
	t.Helper()

	state, err := r.cycle.docker.GetContainerState(t.Context(), r.containerName, instancePublishedPort)
	require.NoError(t, err)
	require.True(t, state.IsRunning, "container %s runs", r.containerName)

	return fmt.Sprintf("http://127.0.0.1:%d", state.HostPort)
}

func (r runningInstance) newClient(t *testing.T) *apiclient.ClientWithResponses {
	t.Helper()

	return apiclient.New(t, r.getURL(t))
}

func (r runningInstance) newSessionClient(t *testing.T, token string) *apiclient.ClientWithResponses {
	t.Helper()

	return apiclient.New(t, r.getURL(t), apiclient.WithSessionToken(token))
}

func (r runningInstance) waitForHealth(t *testing.T) {
	t.Helper()

	exitCode := -1

	err := testwait.Within(containerStartLimit).Poll(t, r.containerName+" answers its health check",
		func(ctx context.Context) error {
			state, err := r.cycle.docker.GetContainerState(ctx, r.containerName, instancePublishedPort)
			if err != nil {
				return err
			}

			if !state.IsRunning {
				exitCode = state.ExitCode

				return nil
			}

			return checkHealth(ctx, fmt.Sprintf("http://127.0.0.1:%d", state.HostPort))
		})
	if err == nil && exitCode >= 0 {
		err = fmt.Errorf("%w: %s exited with code %d during start-up", errInstanceExited, r.containerName, exitCode)
	}

	if err != nil {
		t.Fatal(r.cycle.docker.WithLogTail(context.WithoutCancel(t.Context()), r.containerName, err))
	}
}

func checkHealth(ctx context.Context, baseURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/system/health", nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the health check answered %d", response.StatusCode)
	}

	return nil
}

func (r runningInstance) waitForExit(t *testing.T) exitedInstance {
	t.Helper()

	exitCode, err := r.cycle.docker.WaitForExit(t.Context(), r.containerName, containerStartLimit)
	require.NoError(t, err)

	return exitedInstance{exitCode: exitCode, logs: r.getLogs(t)}
}

func (r runningInstance) stop(t *testing.T) {
	t.Helper()

	require.NoError(t, r.cycle.docker.StopContainer(t.Context(), r.containerName))
}

func (r runningInstance) getLogs(t *testing.T) string {
	t.Helper()

	logs, err := r.cycle.docker.GetLogs(t.Context(), r.containerName)
	require.NoError(t, err)

	return logs
}

func (r runningInstance) getState(t *testing.T) containers.ContainerState {
	t.Helper()

	state, err := r.cycle.docker.GetContainerState(t.Context(), r.containerName, instancePublishedPort)
	require.NoError(t, err)

	return state
}

// queryInternalDatabase runs SQL on the internal database through the connection the container
// published for its own processes, for state the API does not return.
func (r runningInstance) queryInternalDatabase(t *testing.T, statement string) string {
	t.Helper()

	output, err := r.cycle.docker.Execute(t.Context(), r.containerName, []string{
		"sh",
		"-c",
		`/usr/lib/postgresql/17/bin/psql "$(cat /dev/shm/databasus-database-dsn)" -AtXq -v ON_ERROR_STOP=1 -c "$1"`,
		"query",
		statement,
	})
	require.NoError(t, err)

	return strings.TrimSpace(output)
}

// executeAsRuntimeUser runs a shell command in the container as the identity Databasus runs as.
func (r runningInstance) executeAsRuntimeUser(t *testing.T, command string) {
	t.Helper()

	_, err := r.cycle.docker.Execute(t.Context(), r.containerName, []string{"gosu", "databasus", "sh", "-ec", command})
	require.NoError(t, err)
}

// executeAsRoot runs a shell command in the container as root, which can make a directory
// inaccessible to Databasus.
func (r runningInstance) executeAsRoot(t *testing.T, command string) {
	t.Helper()

	_, err := r.cycle.docker.Execute(t.Context(), r.containerName, []string{"sh", "-ec", command})
	require.NoError(t, err)
}

func (r runningInstance) requireRunning(t *testing.T) {
	t.Helper()

	if !r.getState(t).IsRunning {
		t.Fatal(r.cycle.docker.WithLogTail(t.Context(), r.containerName, errInstanceExited))
	}
}
