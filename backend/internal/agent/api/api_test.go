package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/agentclient"
)

func TestClaimVerification(t *testing.T) {
	t.Run("sends nested max ram mb and deserializes when the server returns 200", func(t *testing.T) {
		var capturedPath, capturedAuth string
		var capturedBody map[string]any

		verificationID := uuid.New()
		backupID := uuid.New()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			capturedAuth = r.Header.Get("Authorization")
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"verificationId":     verificationID.String(),
				"backupId":           backupID.String(),
				"backupSizeMb":       120.5,
				"maxContainerDiskMb": 800.25,
				"database": map[string]any{
					"type":              "POSTGRES_LOGICAL",
					"postgresqlLogical": map[string]any{"version": "16"},
				},
			})
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "tok123",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		assignment, err := client.ClaimVerification(t.Context(), agentclient.AgentCapacity{
			MaxCPU: 8, MaxRAMMb: 4096, MaxDiskGb: 100, MaxConcurrentJobs: 4,
		})

		require.NoError(t, err)
		require.NotNil(t, assignment)

		assert.Equal(t, "/api/v1/agent/verifications/11111111-1111-1111-1111-111111111111/claim", capturedPath)
		assert.Equal(t, "Bearer tok123", capturedAuth)

		capacity, ok := capturedBody["capacity"].(map[string]any)
		require.True(t, ok, "claim body must nest capacity under \"capacity\"")
		assert.InDelta(t, float64(4096), capacity["maxRamMb"], 0.001)

		assert.Equal(t, verificationID, assignment.VerificationID)
		assert.Equal(t, backupID, assignment.BackupID)
		assert.InDelta(t, 120.5, assignment.BackupSizeMb, 0.001)
		assert.InDelta(t, 800.25, assignment.MaxContainerDiskMb, 0.001)
		assert.Equal(t, "POSTGRES_LOGICAL", assignment.Database.Type)
		require.NotNil(t, assignment.Database.PostgresqlLogical)
		assert.Equal(t, "16", assignment.Database.PostgresqlLogical.Version)
	})
	t.Run("returns no assignment or error when the server returns 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "t",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		assignment, err := client.ClaimVerification(t.Context(), agentclient.AgentCapacity{})

		require.NoError(t, err)
		assert.Nil(t, assignment)
	})
	t.Run("returns retryable response error when the server returns 500", func(t *testing.T) {
		var calls atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "t",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		_, err := client.ClaimVerification(t.Context(), agentclient.AgentCapacity{})

		require.Error(t, err)
		assert.Equal(t, int32(1), calls.Load(), "claim must not retry internally — the loop owns retry")

		var respErr *ResponseError
		require.ErrorAs(t, err, &respErr)
		assert.True(t, respErr.IsRetryable())
	})
}

func TestDownloadBackup(t *testing.T) {
	t.Run("returns body when the server returns 200", func(t *testing.T) {
		payload := []byte("PGDMP fake custom-format archive")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t,
				"/api/v1/agent/verifications/11111111-1111-1111-1111-111111111111/"+
					"22222222-2222-2222-2222-222222222222/backup-stream",
				r.URL.Path)
			_, _ = w.Write(payload)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		body, err := client.DownloadBackup(
			t.Context(), uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = body.Close() })

		read, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, payload, read)
	})
	t.Run("returns gone response error when the server returns 410", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write([]byte(`{"reason":"gone"}`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		_, err := client.DownloadBackup(t.Context(), uuid.New())

		require.Error(t, err)
		var respErr *ResponseError
		require.ErrorAs(t, err, &respErr)
		assert.True(t, respErr.IsGone())
		assert.False(t, respErr.IsRetryable())
	})
}

func TestReport(t *testing.T) {
	t.Run("succeeds when the server returns 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(
				t,
				"/api/v1/agent/verifications/11111111-1111-1111-1111-111111111111/33333333-3333-3333-3333-333333333333/report",
				r.URL.Path,
			)
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		err := client.Report(t.Context(),
			uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED})

		require.NoError(t, err)
	})
	t.Run("returns err report gone without retry when the server returns 410", func(t *testing.T) {
		var calls atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusGone)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		err := client.Report(
			t.Context(),
			uuid.New(),
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusFAILED},
		)

		require.ErrorIs(t, err, ErrReportGone)
		assert.Equal(t, int32(1), calls.Load(), "410 must not be retried")
	})
	t.Run("retries then succeeds when the server recovers after 5xx errors", func(t *testing.T) {
		var calls atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)
		client.timeAfterFn = func(time.Duration) <-chan time.Time {
			ready := make(chan time.Time, 1)
			ready <- time.Now()
			return ready
		}

		err := client.Report(
			t.Context(),
			uuid.New(),
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusCOMPLETED},
		)

		require.NoError(t, err)
		assert.Equal(t, int32(3), calls.Load())
	})
	t.Run("returns budget error when 5xx errors exhaust the retry budget", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		client.reportRetryBudget = 25 * time.Millisecond

		err := client.Report(
			t.Context(),
			uuid.New(),
			agentclient.VerificationReport{Status: agentclient.VerificationReportStatusFAILED},
		)

		require.ErrorIs(t, err, ErrReportBudgetExhausted)
	})
}

func TestConstants(t *testing.T) {
	t.Run("satisfy invariants", func(t *testing.T) {
		// reportRetryBudget must stay a short, bounded window: once it is exhausted
		// the backend reclaims the run on the agent's next heartbeat, so retrying
		// for minutes would be pointless. 5m is a generous sanity ceiling.
		client := NewClient(Deps{Host: "", Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})
		assert.Less(t, client.reportRetryBudget, 5*time.Minute)

		assert.Equal(t, maxBackoff, reportRetryDelay(99), "report backoff must cap at maxBackoff")
		assert.Equal(t, 1*time.Second, reportRetryDelay(1))
		assert.Equal(t, 2*time.Second, reportRetryDelay(2))
	})
}

func TestHeartbeat(t *testing.T) {
	t.Run("sends flat envelope with bearer and agent path when called", func(t *testing.T) {
		var capturedPath, capturedMethod, capturedAuth string
		var capturedBody map[string]any

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			capturedMethod = r.Method
			capturedAuth = r.Header.Get("Authorization")

			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody)) {
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

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "tok123",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		response, err := client.Heartbeat(t.Context(), agentclient.HeartbeatRequest{
			MaxCPU:                 8,
			MaxRAMMb:               512,
			MaxDiskGb:              100,
			MaxConcurrentJobs:      4,
			CurrentVerificationIDs: []uuid.UUID{},
		})

		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, http.MethodPost, capturedMethod)
		assert.Equal(t,
			"/api/v1/agent/verification/11111111-1111-1111-1111-111111111111/heartbeat",
			capturedPath)
		assert.Equal(t, "Bearer tok123", capturedAuth)

		assert.InDelta(t, float64(8), capturedBody["maxCpu"], 0.001)
		assert.InDelta(t, float64(512), capturedBody["maxRamMb"], 0.001)
		assert.InDelta(t, float64(100), capturedBody["maxDiskGb"], 0.001)
		assert.InDelta(t, float64(4), capturedBody["maxConcurrentJobs"], 0.001)

		require.Contains(t, capturedBody, "currentVerificationIds")
		assert.Equal(t, []any{}, capturedBody["currentVerificationIds"],
			"currentVerificationIds must serialize as [] not null")
	})
	t.Run("returns error without retry when the server returns 4xx", func(t *testing.T) {
		var calls int

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid agent credentials"}`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "bad",
				AgentID: "11111111-1111-1111-1111-111111111111",
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		_, err := client.Heartbeat(t.Context(), agentclient.HeartbeatRequest{CurrentVerificationIDs: []uuid.UUID{}})

		require.Error(t, err)
		assert.Equal(t, 1, calls, "4xx must not be retried")
	})
}

func TestFetchServerVersion(t *testing.T) {
	t.Run("returns version when the server responds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/system/version", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "v3.27.0"})
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})

		version, err := client.FetchServerVersion(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "v3.27.0", version)
	})
	t.Run("sends no cache request headers when called", func(t *testing.T) {
		var requestHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestHeaders = r.Header.Clone()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"v3.45.0"}`))
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})

		_, err := client.FetchServerVersion(t.Context())
		require.NoError(t, err)

		assert.Equal(t, "no-cache", requestHeaders.Get("Cache-Control"))
		assert.Equal(t, "no-cache", requestHeaders.Get("Pragma"))
	})
}

func TestDownloadVerificationAgentBinary(t *testing.T) {
	t.Run("writes it to disk when the server serves a file", func(t *testing.T) {
		payload := []byte("\x7fELF fake verification agent")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/system/verification-agent", r.URL.Path)
			assert.Equal(t, "arm64", r.URL.Query().Get("arch"))
			w.Header().Set(agentVersionHeader, "v3.45.0")
			_, _ = w.Write(payload)
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})
		destPath := filepath.Join(t.TempDir(), "agent")

		downloadedVersion, err := client.DownloadVerificationAgentBinary(
			t.Context(), "arm64", destPath, "v3.43.0",
		)
		require.NoError(t, err)
		assert.Equal(t, "v3.45.0", downloadedVersion)

		written, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, payload, written)
	})
	t.Run("returns fallback version when the server omits the version header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("binary"))
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})

		downloadedVersion, err := client.DownloadVerificationAgentBinary(
			t.Context(), "amd64", filepath.Join(t.TempDir(), "agent"), "v3.43.0",
		)

		require.NoError(t, err)
		assert.Equal(t, "v3.43.0", downloadedVersion)
	})
	t.Run("sends no cache request headers when called", func(t *testing.T) {
		var requestHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestHeaders = r.Header.Clone()
			_, _ = w.Write([]byte("binary"))
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})

		_, err := client.DownloadVerificationAgentBinary(
			t.Context(), "amd64", filepath.Join(t.TempDir(), "agent"), "v1.0.0",
		)
		require.NoError(t, err)

		assert.Equal(t, "no-cache", requestHeaders.Get("Cache-Control"))
		assert.Equal(t, "no-cache", requestHeaders.Get("Pragma"))
	})
	t.Run("returns error when the server returns 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(server.Close)

		client := NewClient(Deps{Host: server.URL, Token: "", AgentID: "", Logger: slog.New(slog.DiscardHandler)})

		_, err := client.DownloadVerificationAgentBinary(
			t.Context(), "amd64", filepath.Join(t.TempDir(), "a"), "v1.0.0",
		)

		require.Error(t, err)
	})
}

func TestHeartbeatRetries(t *testing.T) {
	t.Run("replays the capacity and authentication when the server recovers after two failures", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			assert.Equal(t, "Bearer agent-token", request.Header.Get("Authorization"))
			assert.Equal(t, "no-cache", request.Header.Get("Cache-Control"))
			assert.Equal(t, "no-cache", request.Header.Get("Pragma"))

			var heartbeat agentclient.HeartbeatRequest
			if !assert.NoError(t, json.NewDecoder(request.Body).Decode(&heartbeat)) {
				return
			}
			assert.Equal(t, 512, heartbeat.MaxRAMMb)

			if calls.Add(1) < 3 {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			writer.Header().Set("Content-Type", "application/json")
			assert.NoError(
				t,
				json.NewEncoder(writer).
					Encode(agentclient.HeartbeatResponse{LastSeenAt: time.Now().UTC(), AbortVerificationIDs: []uuid.UUID{}}),
			)
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "agent-token",
				AgentID: uuid.NewString(),
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		response, err := client.Heartbeat(
			t.Context(),
			agentclient.HeartbeatRequest{MaxRAMMb: 512, CurrentVerificationIDs: []uuid.UUID{}},
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, int32(3), calls.Load())
	})
}

func TestDownloadBackupStreaming(t *testing.T) {
	t.Run("returns the first bytes before the server finishes the backup", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/octet-stream")
			_, _ = writer.Write([]byte("PGDMP"))
			assert.NoError(t, http.NewResponseController(writer).Flush())
			<-request.Context().Done()
		}))
		t.Cleanup(server.Close)

		client := NewClient(
			Deps{
				Host:    server.URL,
				Token:   "agent-token",
				AgentID: uuid.NewString(),
				Logger:  slog.New(slog.DiscardHandler),
			},
		)

		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		stream, err := client.DownloadBackup(ctx, uuid.New())
		require.NoError(t, err)
		defer func() { require.NoError(t, stream.Close()) }()

		signature := make([]byte, 5)
		_, err = io.ReadFull(stream, signature)
		require.NoError(t, err)
		assert.Equal(t, "PGDMP", string(signature))
	})
}
