package e2e_test

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	agent_container "databasus-backend/internal/agent/container"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testimages"
)

const pinnedVerificationImageRepository = "databasus/verification-postgres"

// newVerificationImageRepository names a repository for one agent. The agent
// removes superseded tags of its repository at startup, and the Docker daemon
// is shared with runs from other worktrees that may pin other versions, so no
// two agents may clean the same repository.
func newVerificationImageRepository() string {
	return "databasus-e2e-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16] + "/verification-postgres"
}

func getPinnedPostgresVersion(t *testing.T, major string) string {
	t.Helper()

	pinnedVersions, err := agent_container.LoadPinnedPostgresVersions()
	require.NoError(t, err)

	pinnedVersion, isPinned := pinnedVersions.GetPinnedPostgresVersion(major)
	require.True(t, isPinned, "no pinned PostgreSQL version for major %s", major)

	return pinnedVersion
}

// getPinnedVerificationImage returns the test-image catalog's image of the pinned version,
// addressed by digest, so a run never depends on what a registry tag points to.
func getPinnedVerificationImage(t *testing.T, major string) string {
	t.Helper()

	return containers.GetPinnedImage(t, pinnedVerificationImageRepository+":"+getPinnedPostgresVersion(t, major))
}

// provideVerificationImage tags the image of the pinned version under the agent's repository, so the
// agent finds its restore image locally instead of pulling a tag. The image is the test-image
// catalog's record, addressed by digest. A pinned version without a record has no published image
// yet, so its image is built from the recipe, see docs/how-to-upgrade-deps.md.
func (a *agentProcess) provideVerificationImage(t *testing.T, engine *client.Client, major string) string {
	t.Helper()

	pinnedVersion := getPinnedPostgresVersion(t, major)
	agentImage := a.verificationImageRepository + ":" + pinnedVersion
	pinnedImage, err := testimages.GetReference(testimages.Selection{
		Image:        pinnedVerificationImageRepository + ":" + pinnedVersion,
		Architecture: runtime.GOARCH,
	})
	switch {
	case errors.Is(err, testimages.ErrUnknownImage):
		buildVerificationImage(t, agentImage, major, pinnedVersion)
		removeImageOnCleanup(t, engine, agentImage)
	case err != nil:
		t.Fatalf("select the verification image of PostgreSQL %s: %v", pinnedVersion, err)
	default:
		ensureLocalImage(t, engine, pinnedImage)
		tagImage(t, engine, pinnedImage, agentImage)
	}

	return agentImage
}

// buildVerificationImage uses docker buildx because it skips the recipe's unused base stages, which
// the Engine API's classic builder would pull one by one.
func buildVerificationImage(t *testing.T, image, major, pinnedVersion string) {
	t.Helper()

	build := exec.CommandContext(
		t.Context(),
		"docker", "buildx", "build", "--load",
		"--tag", image,
		"--build-arg", "PG_MAJOR="+major,
		"--build-arg", "PG_SERVER_VERSION="+pinnedVersion,
		"../../../cmd/verification-agent/images/postgres",
	)
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
}

func ensureLocalImage(t *testing.T, engine *client.Client, image string) {
	t.Helper()

	if _, err := engine.ImageInspect(t.Context(), image); err == nil {
		return
	}

	download, err := engine.ImagePull(t.Context(), image, client.ImagePullOptions{})
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, download)
	closeErr := download.Close()
	require.NoError(t, err)
	require.NoError(t, closeErr)
}

// tagImage adds a tag and removes it when the test ends. Removing a tag never
// deletes the image while its digest reference remains, so other runs on the
// shared daemon keep it.
func tagImage(t *testing.T, engine *client.Client, sourceImage, targetImage string) {
	t.Helper()

	_, err := engine.ImageTag(t.Context(), client.ImageTagOptions{Source: sourceImage, Target: targetImage})
	require.NoError(t, err)

	removeImageOnCleanup(t, engine, targetImage)
}

func removeImageOnCleanup(t *testing.T, engine *client.Client, image string) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()

		_, _ = engine.ImageRemove(ctx, image, client.ImageRemoveOptions{Force: false})
	})
}

func hasLocalImage(t *testing.T, engine *client.Client, image string) bool {
	t.Helper()

	_, err := engine.ImageInspect(t.Context(), image)

	return err == nil
}
