package container

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSpec(t *testing.T) {
	t.Run("applies hardening controls", func(t *testing.T) {
		containerManager := newTestManager(t)

		spec := containerManager.buildSpec(newTestSpawnPlan())

		assert.True(t, spec.CapDropAll)
		assert.Equal(
			t,
			"postgres",
			spec.User,
			"the container starts unprivileged, so the entrypoint never switches users",
		)
		assert.EqualValues(t, containerPidsLimit, spec.PidsLimit)
		assert.Equal(t, "net-id", spec.NetworkID)
		assert.Equal(t, int64(2)*1_000_000_000, spec.NanoCPUs)
		assert.Equal(t, int64(1024)*1024*1024, spec.MemoryBytes)
		assert.Equal(t, []string{"POSTGRES_PASSWORD=pw"}, spec.Env)
	})
	t.Run("omits cpu limit and keeps other controls when the host does not support CPU limits", func(t *testing.T) {
		containerManager := newTestManager(t)
		containerManager.isCPULimitSupported = false

		spec := containerManager.buildSpec(newTestSpawnPlan())

		assert.Zero(t, spec.NanoCPUs, "Docker treats 0 as no CPU limit")
		assert.Equal(t, int64(1024)*1024*1024, spec.MemoryBytes)
		assert.EqualValues(t, containerPidsLimit, spec.PidsLimit)
		assert.True(t, spec.CapDropAll)
		assert.Equal(t, "postgres", spec.User)
	})
	t.Run("sets restore tuned postgres cmd", func(t *testing.T) {
		containerManager := newTestManager(t)

		spec := containerManager.buildSpec(newTestSpawnPlan())

		assert.Equal(t, getRestoreTunedPostgresCmd(), spec.Cmd)
		assert.Equal(t, "postgres", spec.Cmd[0])
	})
}

func TestGetInContainerConn(t *testing.T) {
	t.Run("uses internal port", func(t *testing.T) {
		c := &PostgresContainer{password: "pw"}

		conn := c.GetInContainerConn()

		assert.Equal(t, "127.0.0.1", conn.Host)
		assert.Equal(t, pgInternalPort, conn.Port)
		assert.Equal(t, restoreUser, conn.User)
		assert.Equal(t, restoreDB, conn.Database)
		assert.Equal(t, "pw", conn.Password)
	})
}

func TestGetVerifierConn(t *testing.T) {
	t.Run("uses resolved host port when a container is created", func(t *testing.T) {
		c := &PostgresContainer{password: "pw", hostPort: 54321}

		conn := c.GetVerifierConn()

		assert.Equal(t, "127.0.0.1", conn.Host)
		assert.Equal(t, 54321, conn.Port)
		assert.Equal(t, restoreUser, conn.User)
		assert.Equal(t, restoreDB, conn.Database)
		assert.Equal(t, "pw", conn.Password)
	})
}

// newTestManager builds a manager without an engine: buildSpec is pure and
// never calls it.
func newTestManager(t *testing.T) *Manager {
	t.Helper()

	containerManager, err := NewManager(
		Deps{
			Engine:          nil,
			AgentID:         "agent-1",
			EngineImageRepo: "databasus/verification-postgres",
			Logger:          slog.New(slog.DiscardHandler),
		},
	)
	require.NoError(t, err)

	return containerManager
}

func newTestSpawnPlan() spawnPlan {
	return spawnPlan{
		verificationID: uuid.New(),
		image:          "postgres@sha256:x",
		password:       "pw",
		cpuPerJob:      2,
		ramMbPerJob:    1024,
		networkID:      "net-id",
		labels:         map[string]string{LabelAgentID: "agent-1"},
	}
}
