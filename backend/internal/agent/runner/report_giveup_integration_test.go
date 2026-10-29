package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/heartbeat"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/lib/testwait"
)

type giveupMockBackend struct {
	mu               sync.Mutex
	heartbeatCount   int
	lastHeartbeatIDs []uuid.UUID

	reportHits       atomic.Int32
	secondReport     chan struct{}
	signalSecondOnce sync.Once
}

func newGiveupMockBackend() (*giveupMockBackend, *httptest.Server) {
	backend := &giveupMockBackend{secondReport: make(chan struct{})}

	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /api/v1/agent/verifications/{agentId}/{id}/backup-stream",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ARCHIVE"))
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verifications/{agentId}/{id}/report",
		func(w http.ResponseWriter, _ *http.Request) {
			if backend.reportHits.Add(1) == 2 {
				backend.signalSecondOnce.Do(func() { close(backend.secondReport) })
			}

			w.WriteHeader(http.StatusServiceUnavailable)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verification/{agentId}/heartbeat",
		func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				CurrentVerificationIDs []uuid.UUID `json:"currentVerificationIds"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			backend.mu.Lock()
			backend.heartbeatCount++
			backend.lastHeartbeatIDs = body.CurrentVerificationIDs
			backend.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC(),
				"abortVerificationIds": []uuid.UUID{},
			})
		},
	)

	return backend, httptest.NewServer(mux)
}

func (b *giveupMockBackend) heartbeatsSeen() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.heartbeatCount
}

func (b *giveupMockBackend) lastHeartbeatVerificationIDs() []uuid.UUID {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.lastHeartbeatIDs
}

func TestExecuteJobReportGiveup(t *testing.T) {
	t.Run("unregisters and heartbeat omits job when the report retry budget is exhausted", func(t *testing.T) {
		backend, server := newGiveupMockBackend()
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{Host: server.URL, Token: "", AgentID: uuid.NewString(), Logger: slog.New(slog.DiscardHandler)},
		)
		heartbeater := heartbeat.NewHeartbeater(client, testCapacity(), slog.New(slog.DiscardHandler))

		runnerUnderTest := NewRunner(
			Deps{
				APIClient: client,
				Capacity:  testCapacity(),
				Pool:      NewPool(2),
				Spawner:   okSpawner(),
				Restorer:  &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}},
				StatsCollector: &fakeStats{stats: verifier.Stats{
					DBSizeBytes: 9_000_000,
					SchemaCount: 2,
					TableCount:  3,
					TableStats:  []verifier.TableStat{{SchemaName: "public", Name: "t1", RowCount: 10}},
				}},
				Heartbeat: heartbeater,
				Logger:    slog.New(slog.DiscardHandler),
				Group:     newTestGroup(t),
			},
		)
		runnerUnderTest.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		job := postgresJob()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go func() {
			select {
			case <-backend.secondReport:
				cancel()
			case <-ctx.Done():
			}
		}()

		runnerUnderTest.executeJob(ctx, job)

		assert.GreaterOrEqual(t, backend.reportHits.Load(), int32(2),
			"the real report retry loop must have actually retried against the unresponsive server")
		assert.NotContains(t, heartbeater.GetRunningVerificationIDs(), job.VerificationID,
			"report give-up must untrack the job from the real heartbeat registry")

		beatCtx, beatCancel := context.WithCancel(t.Context())
		beatDone := make(chan struct{})
		go func() {
			heartbeater.Run(beatCtx)
			close(beatDone)
		}()

		testwait.Until(t, "heartbeat sent", func(context.Context) error {
			if backend.heartbeatsSeen() == 0 {
				return fmt.Errorf("no heartbeat received")
			}
			return nil
		})

		beatCancel()
		<-beatDone

		assert.NotContains(t, backend.lastHeartbeatVerificationIDs(), job.VerificationID,
			"after give-up the real heartbeat wire must no longer carry the dropped job")
	})
}
