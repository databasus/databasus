package heartbeat

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/config"
)

func TestBeat(t *testing.T) {
	t.Run(
		"reports capacity in megabytes when no verifications are registered",
		func(t *testing.T) {
			var body map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
					"abortVerificationIds": []string{},
				})
			}))
			t.Cleanup(server.Close)

			client := api.NewClient(
				api.Deps{
					Host:    server.URL,
					Token:   "tok",
					AgentID: "11111111-1111-1111-1111-111111111111",
					Logger:  slog.New(slog.DiscardHandler),
				},
			)
			capacity := config.Capacity{
				MaxCPU:            8,
				MaxRAMMb:          512,
				MaxDiskGb:         100,
				MaxConcurrentJobs: 4,
			}

			NewHeartbeater(
				client,
				capacity,
				slog.New(slog.DiscardHandler),
			).sendHeartbeat(t.Context(), slog.New(slog.DiscardHandler))

			assert.InDelta(t, float64(8), body["maxCpu"], 0.001)
			assert.InDelta(t, float64(512), body["maxRamMb"], 0.001, "configured memory must be reported unchanged")
			assert.InDelta(t, float64(100), body["maxDiskGb"], 0.001)
			assert.InDelta(t, float64(4), body["maxConcurrentJobs"], 0.001)
			assert.Equal(t, []any{}, body["currentVerificationIds"])
		},
	)
	t.Run("reports its id as current verification id when a job is registered", func(t *testing.T) {
		var body map[string]any

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
				"abortVerificationIds": []string{},
			})
		}))
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{
				Host:    server.URL,
				Token:   "tok",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)
		hb := NewHeartbeater(client, config.Capacity{}, slog.New(slog.DiscardHandler))

		jobID := uuid.New()
		hb.TrackVerification(jobID, func() {})

		hb.sendHeartbeat(t.Context(), slog.New(slog.DiscardHandler))

		ids, ok := body["currentVerificationIds"].([]any)
		require.True(t, ok)
		require.Len(t, ids, 1)
		assert.Equal(t, jobID.String(), ids[0])
	})
	t.Run("cancels matching job and records abort when the response lists an abort ID", func(t *testing.T) {
		abortID := uuid.New()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
				"abortVerificationIds": []string{abortID.String()},
			})
		}))
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{
				Host:    server.URL,
				Token:   "tok",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)
		hb := NewHeartbeater(client, config.Capacity{}, slog.New(slog.DiscardHandler))

		canceled := make(chan struct{})
		hb.TrackVerification(abortID, func() { close(canceled) })

		hb.sendHeartbeat(t.Context(), slog.New(slog.DiscardHandler))

		select {
		case <-canceled:
		case <-time.After(2 * time.Second):
			t.Fatal("registered job's context was not cancelled on server abort")
		}

		assert.True(t, hb.IsAborted(abortID),
			"the abort set must be recorded for the runner's pre-FAILED re-check")
	})
}

func TestHeartbeaterRun(t *testing.T) {
	t.Run("panics when Run is called twice", func(t *testing.T) {
		client := api.NewClient(
			api.Deps{
				Host:    "http://127.0.0.1:0",
				Token:   "tok",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)
		heartbeater := NewHeartbeater(client, config.Capacity{}, slog.New(slog.DiscardHandler))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		heartbeater.Run(ctx)

		assert.Panics(t, func() { heartbeater.Run(ctx) })
	})
	t.Run("stops promptly when the context is cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
				"abortVerificationIds": []string{},
			})
		}))
		t.Cleanup(server.Close)

		client := api.NewClient(
			api.Deps{
				Host:    server.URL,
				Token:   "tok",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)
		heartbeater := NewHeartbeater(client, config.Capacity{MaxConcurrentJobs: 1}, slog.New(slog.DiscardHandler))

		ctx, cancel := context.WithCancel(t.Context())
		stopped := make(chan struct{})

		go func() {
			heartbeater.Run(ctx)
			close(stopped)
		}()

		cancel()

		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			t.Fatal("Run did not return after context cancellation")
		}
	})
}
