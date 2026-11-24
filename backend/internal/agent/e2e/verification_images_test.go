package e2e_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	agent_container "databasus-backend/internal/agent/container"
	containers "databasus-backend/internal/lib/testcontainers"
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

// getPinnedVerificationImage returns the catalog image of the pinned version,
// addressed by digest, so a run never depends on what a registry tag points to.
func getPinnedVerificationImage(t *testing.T, major string) string {
	t.Helper()

	return containers.GetPinnedImage(t, pinnedVerificationImageRepository+":"+getPinnedPostgresVersion(t, major))
}

// provideVerificationImage tags the pinned catalog image under the agent's
// repository, so the agent finds its restore image locally instead of pulling
// a tag that exists only after a release publishes it.
func (a *agentProcess) provideVerificationImage(t *testing.T, engine *client.Client, major string) string {
	t.Helper()

	pinnedImage := getPinnedVerificationImage(t, major)
	ensureLocalImage(t, engine, pinnedImage)

	agentImage := a.verificationImageRepository + ":" + getPinnedPostgresVersion(t, major)
	tagImage(t, engine, pinnedImage, agentImage)

	return agentImage
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

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()

		_, _ = engine.ImageRemove(ctx, targetImage, client.ImageRemoveOptions{Force: false})
	})
}

func hasLocalImage(t *testing.T, engine *client.Client, image string) bool {
	t.Helper()

	_, err := engine.ImageInspect(t.Context(), image)

	return err == nil
}
