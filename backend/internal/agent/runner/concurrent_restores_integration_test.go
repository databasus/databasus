package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/heartbeat"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/testwait"
)

type concurrentRestoreBackend struct {
	claims   []*agentclient.VerificationAssignment
	claimIdx atomic.Int32

	mu      sync.Mutex
	reports map[uuid.UUID]agentclient.VerificationReportStatus
}

func newConcurrentRestoreBackend(
	claims []*agentclient.VerificationAssignment,
) (*concurrentRestoreBackend, *httptest.Server) {
	backend := &concurrentRestoreBackend{
		claims:  claims,
		reports: make(map[uuid.UUID]agentclient.VerificationReportStatus),
	}

	mux := http.NewServeMux()

	mux.HandleFunc(
		"POST /api/v1/agent/verifications/{agentId}/claim",
		func(w http.ResponseWriter, _ *http.Request) {
			i := int(backend.claimIdx.Add(1)) - 1
			if i >= len(backend.claims) {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(backend.claims[i])
		},
	)

	mux.HandleFunc(
		"GET /api/v1/agent/verifications/{agentId}/{id}/backup-stream",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ARCHIVE"))
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verifications/{agentId}/{id}/report",
		func(w http.ResponseWriter, req *http.Request) {
			verificationID, err := uuid.Parse(req.PathValue("id"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var body struct {
				Status agentclient.VerificationReportStatus `json:"status"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			backend.mu.Lock()
			backend.reports[verificationID] = body.Status
			backend.mu.Unlock()

			w.WriteHeader(http.StatusNoContent)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verification/{agentId}/heartbeat",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC(),
				"abortVerificationIds": []uuid.UUID{},
			})
		},
	)

	return backend, httptest.NewServer(mux)
}

func (b *concurrentRestoreBackend) reportsSnapshot() map[uuid.UUID]agentclient.VerificationReportStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make(map[uuid.UUID]agentclient.VerificationReportStatus, len(b.reports))
	maps.Copy(out, b.reports)

	return out
}

type recordingContainer struct {
	id              int
	inContainerConn dbconn.Conn
	verifierConn    dbconn.Conn
	terminated      atomic.Bool
}

func (c *recordingContainer) Exec(
	_ context.Context, _ []string, stdin io.Reader, _ []string,
) (restore.ExecResult, error) {
	if stdin != nil {
		_, _ = io.Copy(io.Discard, stdin)
	}

	return restore.ExecResult{}, nil
}

func (c *recordingContainer) GetInContainerConn() dbconn.Conn { return c.inContainerConn }
func (c *recordingContainer) GetVerifierConn() dbconn.Conn    { return c.verifierConn }

func (c *recordingContainer) GetDiskUsageBytes(context.Context) (int64, error) {
	return 0, nil
}

func (c *recordingContainer) Terminate(context.Context) error {
	c.terminated.Store(true)
	return nil
}

type recordingSpawner struct {
	nextID atomic.Int32

	mu         sync.Mutex
	containers []*recordingContainer
}

func (s *recordingSpawner) Spawn(_ context.Context, _ SpawnRequest) (JobContainer, error) {
	c := &recordingContainer{
		id:              int(s.nextID.Add(1)),
		inContainerConn: testConn(),
		verifierConn:    testConn(),
	}

	s.mu.Lock()
	s.containers = append(s.containers, c)
	s.mu.Unlock()

	return c, nil
}

func (s *recordingSpawner) snapshot() []*recordingContainer {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*recordingContainer, len(s.containers))
	copy(out, s.containers)

	return out
}

type barrierRestorer struct {
	barrier *sync.WaitGroup
}

func (r *barrierRestorer) StageBackupViaExec(
	_ context.Context, _ restore.ExecRunner, body io.Reader, _ string,
) error {
	if body != nil {
		_, _ = io.Copy(io.Discard, body)
	}

	return nil
}

func (r *barrierRestorer) RunPgRestore(
	_ context.Context, _ restore.ExecRunner, _ restore.PgRestoreSpec,
) (restore.Result, error) {
	r.barrier.Done()
	r.barrier.Wait()

	return restore.Result{PgRestoreExitCode: 0, DurationMs: 1}, nil
}

func (r *barrierRestorer) EnsureArchiveOwnerRoles(
	_ context.Context, _ restore.ExecRunner, _ string, _ dbconn.Conn,
) ([]string, error) {
	return nil, nil
}

func (r *barrierRestorer) RunTimescalePreRestore(context.Context, dbconn.Conn) error  { return nil }
func (r *barrierRestorer) RunTimescalePostRestore(context.Context, dbconn.Conn) error { return nil }

func makeConcurrentAssignment(id uuid.UUID) *agentclient.VerificationAssignment {
	return &agentclient.VerificationAssignment{
		VerificationID:     id,
		BackupID:           uuid.New(),
		BackupSizeMb:       1,
		MaxContainerDiskMb: 0,
		Database: agentclient.AssignedDatabase{
			Type:              "POSTGRES_LOGICAL",
			PostgresqlLogical: &agentclient.AssignedPostgresql{Version: "16"},
		},
	}
}

func TestRunConcurrentRestores(t *testing.T) {
	t.Run("each gets isolated container and both complete when two verifications are claimed", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		claims := []*agentclient.VerificationAssignment{
			makeConcurrentAssignment(id1),
			makeConcurrentAssignment(id2),
		}

		backend, server := newConcurrentRestoreBackend(claims)
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: uuid.NewString(), Logger: slog.New(slog.DiscardHandler)},
		)
		heartbeater := heartbeat.NewHeartbeater(client, testCapacity(), slog.New(slog.DiscardHandler))

		spawner := &recordingSpawner{}
		barrier := &sync.WaitGroup{}
		barrier.Add(2)

		runnerUnderTest := NewRunner(
			Deps{
				APIClient:      client,
				Capacity:       testCapacity(),
				Pool:           NewPool(2),
				Spawner:        spawner,
				Restorer:       &barrierRestorer{barrier: barrier},
				StatsCollector: &fakeStats{stats: verifier.Stats{DBSizeBytes: 1, SchemaCount: 1, TableCount: 1}},
				Heartbeat:      heartbeater,
				Logger:         slog.New(slog.DiscardHandler),
				Group:          newTestGroup(t),
			},
		)
		runnerUnderTest.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		runCtx, runCancel := context.WithCancel(t.Context())
		runDone := make(chan error, 1)
		go func() {
			runDone <- runnerUnderTest.Run(runCtx)
		}()

		testwait.Until(t, "both verifications complete", func(context.Context) error {
			snapshot := backend.reportsSnapshot()
			if snapshot[id1] != agentclient.VerificationReportStatusCOMPLETED ||
				snapshot[id2] != agentclient.VerificationReportStatusCOMPLETED {
				return fmt.Errorf("verification reports: %v", snapshot)
			}
			return nil
		})

		runCancel()
		select {
		case err := <-runDone:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run did not return after context cancellation")
		}

		spawned := spawner.snapshot()
		require.Len(t, spawned, 2,
			"each claimed verification must spawn its own container — proves isolation, not reuse")
		assert.NotEqual(t, spawned[0].id, spawned[1].id,
			"the two spawned containers must be distinct instances")
		for _, c := range spawned {
			assert.True(t, c.terminated.Load(),
				"every spawned container must be torn down (deferred Terminate in executeJob)")
		}

		reports := backend.reportsSnapshot()
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, reports[id1])
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, reports[id2])
	})
}
