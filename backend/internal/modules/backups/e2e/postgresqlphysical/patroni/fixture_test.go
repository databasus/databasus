package patroni_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestPatroniFixtureLifecycle(t *testing.T) {
	t.Parallel()

	capacity := make(chan struct{}, 2)
	acquire := func(t *testing.T) {
		t.Helper()
		select {
		case capacity <- struct{}{}:
			t.Cleanup(func() { <-capacity })
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}

	for _, mode := range []string{"success", "partial-startup", "cancellation", "transfer-failure", "parent-interruption"} {
		t.Run("removes owned resources after "+mode, func(t *testing.T) {
			t.Parallel()

			acquire(t)
			runPatroniCleanupScenario(t, mode)
		})
	}

	for _, mode := range []string{"transfer-release", "transfer-cancellation"} {
		t.Run("cleans up a backup with "+mode, func(t *testing.T) {
			t.Parallel()

			acquire(t)
			fixture := containers.StartPatroni(t, containers.PatroniSpec{Image: "postgres:17"})
			runPatroniTransferLifecycle(t, fixture, mode)
		})
	}

	for _, image := range []string{"postgres:17", "postgres:18"} {
		t.Run("elects the replica after primary loss with "+image, func(t *testing.T) {
			t.Parallel()

			acquire(t)
			fixture := containers.StartPatroni(t, containers.PatroniSpec{Image: image})
			fixture.RequireCycles(t, fixture.GetNode(fixture.GetIdentity(t).Node), 3)
			fixture.KillPrimary(t)
		})

		t.Run("routes to both nodes after two switchovers with "+image, func(t *testing.T) {
			t.Parallel()

			acquire(t)

			fixture := containers.StartPatroni(t, containers.PatroniSpec{Image: image})
			initial := fixture.GetIdentity(t)

			second := fixture.Switchover(t)
			third := fixture.Switchover(t)

			require.NotEqual(t, initial.Node, second.Node)
			require.Equal(t, initial.Node, third.Node)
			require.Greater(t, third.Timeline, second.Timeline)
		})
	}
}
