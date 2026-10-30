package container

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildSpec(t *testing.T) {
	t.Run("applies hardening controls", func(t *testing.T) {
		// nil engine: buildSpec is pure and never calls the engine.
		containerManager := NewManager(
			Deps{
				Engine:          nil,
				AgentID:         "agent-1",
				EngineImageRepo: "databasus/verification-postgres",
				Logger:          slog.New(slog.DiscardHandler),
			},
		)

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
		containerManager := NewManager(
			Deps{
				Engine:          nil,
				AgentID:         "agent-1",
				EngineImageRepo: "databasus/verification-postgres",
				Logger:          slog.New(slog.DiscardHandler),
			},
		)
		containerManager.isCPULimitSupported = false

		spec := containerManager.buildSpec(newTestSpawnPlan())

		assert.Zero(t, spec.NanoCPUs, "Docker treats 0 as no CPU limit")
		assert.Equal(t, int64(1024)*1024*1024, spec.MemoryBytes)
		assert.EqualValues(t, containerPidsLimit, spec.PidsLimit)
		assert.True(t, spec.CapDropAll)
		assert.Equal(t, "postgres", spec.User)
	})
	t.Run("sets restore tuned postgres cmd", func(t *testing.T) {
		containerManager := NewManager(
			Deps{
				Engine:          nil,
				AgentID:         "agent-1",
				EngineImageRepo: "databasus/verification-postgres",
				Logger:          slog.New(slog.DiscardHandler),
			},
		)

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

func TestImageForMajor(t *testing.T) {
	t.Run("appends major tag to configured repo", func(t *testing.T) {
		cases := []struct {
			major    string
			expected string
		}{
			{"12", "databasus/verification-postgres:12"},
			{"16", "databasus/verification-postgres:16"},
			{"18", "databasus/verification-postgres:18"},
		}

		for _, tc := range cases {
			t.Run(tc.major, func(t *testing.T) {
				assert.Equal(t, tc.expected, imageForMajor(tc.major, "databasus/verification-postgres"))
			})
		}
	})
}

func TestImageForJob(t *testing.T) {
	t.Run("returns version matched timescale image when a TimescaleDB version is set", func(t *testing.T) {
		image := imageForJob(
			SpawnRequest{PgMajor: "17", TimescaledbVersion: "2.17.0"}, "databasus/verification-postgres")

		assert.Equal(t, "timescale/timescaledb:2.17.0-pg17", image,
			"timescale jobs ignore the engine repo and use the version-matched timescale image")
	})
	t.Run("uses configured engine repo when no TimescaleDB version is set", func(t *testing.T) {
		image := imageForJob(SpawnRequest{PgMajor: "17"}, "databasus/verification-postgres")

		assert.Equal(t, "databasus/verification-postgres:17", image)
	})
}

func TestStockFallbackImage(t *testing.T) {
	t.Run("falls back to stock postgres when the bundled image is unavailable", func(t *testing.T) {
		fallbackImage, canFallBack := stockFallbackImage(
			SpawnRequest{PgMajor: "16"}, "databasus/verification-postgres:16")

		assert.True(t, canFallBack)
		assert.Equal(t, "postgres:16", fallbackImage)
	})
	t.Run("does not fall back when the image already uses the stock repository", func(t *testing.T) {
		_, canFallBack := stockFallbackImage(SpawnRequest{PgMajor: "16"}, "postgres:16")

		assert.False(t, canFallBack, "a job already on the stock image has nothing to fall back to")
	})
	t.Run("does not fall back when the job uses TimescaleDB", func(t *testing.T) {
		_, canFallBack := stockFallbackImage(
			SpawnRequest{PgMajor: "16", TimescaledbVersion: "2.17.0"}, "timescale/timescaledb:2.17.0-pg16")

		assert.False(t, canFallBack,
			"timescale needs its exact version-matched image; stock postgres cannot restore it")
	})
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
