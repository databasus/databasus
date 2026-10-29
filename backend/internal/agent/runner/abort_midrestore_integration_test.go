package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/heartbeat"
	"databasus-backend/internal/lib/testwait"
)

type abortMidRestoreBackend struct {
	abortID    uuid.UUID
	reportHits atomic.Int32
}

func newAbortMidRestoreBackend(abortID uuid.UUID) (*abortMidRestoreBackend, *httptest.Server) {
	backend := &abortMidRestoreBackend{abortID: abortID}

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
			backend.reportHits.Add(1)

			w.WriteHeader(http.StatusOK)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verification/{agentId}/heartbeat",
		func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				CurrentVerificationIDs []uuid.UUID `json:"currentVerificationIds"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			abortIDs := []uuid.UUID{}
			if slices.Contains(body.CurrentVerificationIDs, backend.abortID) {
				abortIDs = []uuid.UUID{backend.abortID}
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC(),
				"abortVerificationIds": abortIDs,
			})
		},
	)

	return backend, httptest.NewServer(mux)
}

func TestExecuteJobAbortMidRestore(t *testing.T) {
	t.Run(
		"hard stops restore cleans container and sends no report when the server aborts a running restore",
		func(t *testing.T) {
			job := postgresJob()

			backend, server := newAbortMidRestoreBackend(job.VerificationID)
			t.Cleanup(server.Close)

			client := api.NewClient(
				api.Deps{Host: server.URL, Token: "", AgentID: uuid.NewString(), Logger: slog.New(slog.DiscardHandler)},
			)
			heartbeater := heartbeat.NewHeartbeater(client, testCapacity(), slog.New(slog.DiscardHandler))

			jobContainer := fakeContainerWith(testConn())
			restorer := &fakeRestorer{runBlocks: true}

			runnerUnderTest := NewRunner(
				Deps{
					APIClient:      client,
					Capacity:       testCapacity(),
					Pool:           NewPool(2),
					Spawner:        &fakeSpawner{container: jobContainer},
					Restorer:       restorer,
					StatsCollector: &fakeStats{},
					Heartbeat:      heartbeater,
					Logger:         slog.New(slog.DiscardHandler),
					Group:          newTestGroup(t),
				},
			)

			execDone := make(chan struct{})
			go func() {
				runnerUnderTest.executeJob(t.Context(), job)
				close(execDone)
			}()

			testwait.Until(t, "restore begins", func(context.Context) error {
				if !restorer.runEntered.Load() {
					return fmt.Errorf("restore has not started")
				}
				return nil
			})

			beatCtx, beatCancel := context.WithCancel(t.Context())
			beatDone := make(chan struct{})
			go func() {
				heartbeater.Run(beatCtx)
				close(beatDone)
			}()

			select {
			case <-execDone:
			case <-time.After(5 * time.Second):
				t.Fatal("executeJob did not return after the server abort; restore was not hard-stopped")
			}

			beatCancel()
			<-beatDone

			assert.True(t, restorer.runCtxCancelled.Load(),
				"pg_restore must be hard-stopped by jobCtx cancellation, not run to completion")
			assert.True(t, jobContainer.terminated,
				"the container and its anonymous PGDATA volume must be torn down on a server abort, "+
					"even though the same cancellation triggered the teardown")
			assert.Equal(t, int32(0), backend.reportHits.Load(),
				"a server-aborted verification must be dropped silently with no FAILED report")
			assert.True(t, heartbeater.IsAborted(job.VerificationID),
				"the abort set must be recorded for the runner's pre-FAILED re-check")
		},
	)
}
