package containers

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const patroniImageRepository = "databasus-test-patroni"

// buildPatroniImage builds the node image once at a time across every test process on the machine.
// Parallel tests and their child processes share one tag, and concurrent builds of that tag fail with
// AlreadyExists while a cold build compiles haproxy for minutes. The kernel releases the flock when a
// holder dies, so a killed child cannot leave it held.
func buildPatroniImage(t *testing.T, postgresImage string) string {
	t.Helper()

	cacheDirectory, err := os.UserCacheDir()
	require.NoError(t, err)
	lockDirectory := filepath.Join(cacheDirectory, "databasus-tests")
	require.NoError(t, os.MkdirAll(lockDirectory, 0o700))
	tag := restoreTargetTag(postgresImage)
	lockFile, err := os.OpenFile(
		filepath.Join(lockDirectory, patroniImageRepository+"-"+tag+".lock"),
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	require.NoError(t, err)
	defer func() { _ = lockFile.Close() }()

	waitStartedAt := time.Now()
	require.NoError(t, syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX))
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()
	t.Logf("patroni image %s lock acquired after %s", tag, time.Since(waitStartedAt))

	baseImage := GetPinnedImage(t, postgresImage)
	provider, err := testcontainers.NewDockerProvider()
	require.NoError(t, err)
	defer func() { _ = provider.Close() }()

	buildStartedAt := time.Now()
	image, err := provider.BuildImage(t.Context(), &testcontainers.ContainerRequest{
		Context:   getFixtureDirectory(t, "patroni"),
		Repo:      patroniImageRepository,
		Tag:       tag,
		KeepImage: true,
		BuildArgs: map[string]*string{"POSTGRES_IMAGE": &baseImage},
	})
	require.NoError(t, err, "build Patroni image %s", tag)
	t.Logf("patroni image %s built after %s", image, time.Since(buildStartedAt))

	return image
}
