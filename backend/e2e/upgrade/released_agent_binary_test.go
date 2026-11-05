package upgrade

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

const releasedAgentImage = "databasus/databasus@sha256:a4bbf8b285c54f76c61a997d604cb22d7f53a01fe72b644a8abdb9b5b4cc82a5"

func extractReleasedAgent(t *testing.T, docker *dockerHost) string {
	t.Helper()

	require.NoError(t, docker.ensureImage(t.Context(), releasedAgentImage))

	fixture, err := docker.client.ContainerCreate(t.Context(), client.ContainerCreateOptions{
		Config: &container.Config{Image: releasedAgentImage, Labels: docker.labels},
	})
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
		defer cancel()
		require.NoError(t, docker.removeAgentContainer(ctx, fixture.ID))
	}()

	archive, err := docker.client.CopyFromContainer(t.Context(), fixture.ID, client.CopyFromContainerOptions{
		SourcePath: "/app/agent-binaries/databasus-verification-agent-linux-" + runtime.GOARCH,
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, archive.Content.Close()) }()

	reader := tar.NewReader(archive.Content)
	header, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, byte(tar.TypeReg), header.Typeflag)

	binaryPath := filepath.Join(t.TempDir(), "verification-agent")
	binary, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o755)
	require.NoError(t, err)

	_, copyErr := io.Copy(binary, reader)
	require.NoError(t, binary.Close())
	require.NoError(t, copyErr)

	require.Equal(t, "3.64.0", getAgentBinaryVersion(t, binaryPath))
	t.Logf("released agent fixture: %s, linux/%s, version 3.64.0", releasedAgentImage, runtime.GOARCH)

	return binaryPath
}
