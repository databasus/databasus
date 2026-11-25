package e2e_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestAgentStartup(t *testing.T) {
	t.Parallel()

	t.Run("removes its leftovers while keeping another agent's network", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		admin := app.CreateAdmin(t)
		credentials := app.CreateVerificationAgent(t, admin)
		engine := newEngine(t, credentials.Agent.ID)

		createStrayContainer(t, engine, credentials.Agent.ID)
		createNetwork(t, engine, credentials.Agent.ID, uuid.New())

		otherAgentID := uuid.New()
		otherEngine := newEngine(t, otherAgentID)
		otherNetworkID := createNetwork(t, otherEngine, otherAgentID, uuid.New())

		gate := newGateway(t, app.URL, gatewayOptions{})
		agent := newAgentProcess(t, currentBinaryDirectory)
		agent.start(t, gate.url, credentials)
		waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Time{})
		requireNoLeftovers(t, engine, credentials.Agent.ID)

		networks, err := otherEngine.NetworkList(
			t.Context(),
			client.NetworkListOptions{Filters: agentFilter(otherAgentID)},
		)
		require.NoError(t, err)
		require.Len(t, networks.Items, 1)
		assert.Equal(t, otherNetworkID, networks.Items[0].ID)
	})

	t.Run("removes superseded verification images while keeping pinned, used and unrelated ones", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		admin := app.CreateAdmin(t)
		credentials := app.CreateVerificationAgent(t, admin)
		engine := newEngine(t, credentials.Agent.ID)
		agent := newAgentProcess(t, currentBinaryDirectory)
		repository := agent.verificationImageRepository

		pinnedImage := agent.provideVerificationImage(t, engine, "18")
		sourceImage := getPinnedVerificationImage(t, "18")
		tagImage(t, engine, sourceImage, repository+":18")
		tagImage(t, engine, sourceImage, repository+":18.6")
		usedImage := createImageUsedByContainer(t, engine, sourceImage, repository+":17.11")
		unrelatedImage := newVerificationImageRepository() + ":18.6"
		tagImage(t, engine, sourceImage, unrelatedImage)

		gate := newGateway(t, app.URL, gatewayOptions{})
		agent.start(t, gate.url, credentials)
		waitForHeartbeat(t, app, admin, credentials.Agent.ID, time.Time{})

		assert.False(t, hasLocalImage(t, engine, repository+":18"))
		assert.False(t, hasLocalImage(t, engine, repository+":18.6"))
		assert.True(t, hasLocalImage(t, engine, pinnedImage), "the pinned tag stays on an image that lost another tag")
		assert.True(t, hasLocalImage(t, engine, usedImage), "an image a container uses stays")
		assert.True(t, hasLocalImage(t, engine, unrelatedImage), "another repository is not cleaned")

		output, err := os.ReadFile(agent.outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(output), "could not remove superseded verification image")
	})
}

// createImageUsedByContainer commits a new image under the given tag only and
// creates a container from it, so the tag is the image's last reference and
// Docker refuses to remove it without force.
func createImageUsedByContainer(t *testing.T, engine *client.Client, sourceImage, targetImage string) string {
	t.Helper()

	source, err := engine.ContainerCreate(t.Context(), client.ContainerCreateOptions{
		Image: sourceImage,
		Name:  "databasus-image-source-" + uuid.NewString(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { removeContainer(t, engine, source.ID) })

	committed, err := engine.ContainerCommit(
		t.Context(),
		source.ID,
		client.ContainerCommitOptions{Reference: targetImage},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()

		_, _ = engine.ImageRemove(ctx, committed.ID, client.ImageRemoveOptions{Force: true})
	})

	user, err := engine.ContainerCreate(t.Context(), client.ContainerCreateOptions{
		Image: targetImage,
		Name:  "databasus-image-user-" + uuid.NewString(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { removeContainer(t, engine, user.ID) })

	return targetImage
}

func removeContainer(t *testing.T, engine *client.Client, containerID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
	defer cancel()

	_, _ = engine.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}
