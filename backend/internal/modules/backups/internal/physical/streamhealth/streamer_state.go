package streamhealth

import (
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// A RUNNING row whose heartbeat is past the takeover age may belong to a
// process that is gone, so only a fresh heartbeat counts as running.
func GetStreamerState(streamer *catalog.WalStreamer, now time.Time) StreamerState {
	if streamer == nil {
		return StreamerStateNotStarted
	}

	if streamer.Status == catalog.WalStreamerStatusRunning &&
		now.Sub(streamer.LastHeartbeatAt) < StreamerHeartbeatStaleness {
		return StreamerStateRunning
	}

	return StreamerStateNotRunning
}

// Receiver restarts and whole-session starts both mean streaming was
// interrupted, so they add up.
func GetRestartsLastHour(
	recentSamples []*catalog.WalStreamSample,
	sessionStartsLastHour int,
	now time.Time,
) int {
	restarts := sessionStartsLastHour
	windowStart := now.Add(-frequentRestartsWindow)

	for _, sample := range recentSamples {
		if sample.SampledAt.After(windowStart) {
			restarts += sample.ReceiverRestarts
		}
	}

	return restarts
}
