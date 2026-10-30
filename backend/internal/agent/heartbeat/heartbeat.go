package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/config"
	"databasus-backend/internal/lib/agentclient"
)

const (
	jobName           = "verification_heartbeat"
	heartbeatInterval = 30 * time.Second
)

// Heartbeater periodically reports the agent's capacity and the set of
// in-flight verification IDs (the backend reclaims any RUNNING job absent from
// that set), and acts on the abort list in the response. It is the single
// source of truth for which verifications are active and records the last
// abort set so the runner can re-check it immediately before any FAILED report
// (the abort/report-race mitigation).
type Heartbeater struct {
	api      *api.Client
	capacity config.Capacity
	hasRun   atomic.Bool

	mu        sync.Mutex
	registry  map[uuid.UUID]context.CancelFunc
	lastAbort map[uuid.UUID]struct{}

	log *slog.Logger
}

func NewHeartbeater(apiClient *api.Client, capacity config.Capacity, log *slog.Logger) *Heartbeater {
	return &Heartbeater{
		api:       apiClient,
		capacity:  capacity,
		registry:  make(map[uuid.UUID]context.CancelFunc),
		lastAbort: make(map[uuid.UUID]struct{}),
		log:       log,
	}
}

func (h *Heartbeater) Run(ctx context.Context) {
	if h.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", h))
	}

	logger := h.log.With(slog.Any("job_id", uuid.New()), slog.Any("job_name", jobName))
	logger.InfoContext(ctx, "heartbeat loop started")

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	if ctx.Err() == nil {
		h.sendHeartbeat(ctx, logger)
	}

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "heartbeat loop stopped")

			return
		case <-ticker.C:
			h.sendHeartbeat(ctx, logger)
		}
	}
}

// TrackVerification must be called before the container/network exists so the
// ID rides the heartbeat envelope and is reachable by an abort the instant it
// has any artifact.
func (h *Heartbeater) TrackVerification(id uuid.UUID, cancel context.CancelFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.registry[id] = cancel
}

func (h *Heartbeater) UntrackVerification(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.registry, id)
}

func (h *Heartbeater) GetRunningVerificationIDs() []uuid.UUID {
	h.mu.Lock()
	defer h.mu.Unlock()

	ids := make([]uuid.UUID, 0, len(h.registry))
	for id := range h.registry {
		ids = append(ids, id)
	}

	return ids
}

func (h *Heartbeater) IsAborted(id uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, aborted := h.lastAbort[id]

	return aborted
}

func (h *Heartbeater) sendHeartbeat(ctx context.Context, logger *slog.Logger) {
	request := agentclient.HeartbeatRequest{
		MaxCPU:                 h.capacity.MaxCPU,
		MaxRAMMb:               h.capacity.MaxRAMMb,
		MaxDiskGb:              h.capacity.MaxDiskGb,
		MaxConcurrentJobs:      h.capacity.MaxConcurrentJobs,
		CurrentVerificationIDs: h.GetRunningVerificationIDs(),
	}

	response, err := h.api.Heartbeat(ctx, request)
	if err != nil {
		logger.WarnContext(ctx, "heartbeat failed, will retry next tick", slog.Any("error", err))

		return
	}

	h.abortVerifications(ctx, response.AbortVerificationIDs, logger)

	logger.DebugContext(ctx, "heartbeat ok", slog.Any("last_seen_at", response.LastSeenAt.UTC().Format(time.RFC3339)))
}

func (h *Heartbeater) abortVerifications(ctx context.Context, abortIDs []uuid.UUID, logger *slog.Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastAbort = make(map[uuid.UUID]struct{}, len(abortIDs))
	for _, id := range abortIDs {
		h.lastAbort[id] = struct{}{}

		if cancel, registered := h.registry[id]; registered {
			logger.InfoContext(ctx, "aborting verification on server request", slog.Any("verification_id", id))
			cancel()
		}
	}
}
