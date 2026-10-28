package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamStreamer(streamer physical.WalStreamStreamer) (WalStreamStreamer, error) {
	body := WalStreamStreamer{}
	body.FailingSince = streamer.FailingSince
	body.RestartsLastHour = streamer.RestartsLastHour
	body.SessionStartedAt = streamer.SessionStartedAt

	switch string(streamer.State) {
	case "RUNNING", "NOT_RUNNING", "NOT_STARTED":
	default:
		return WalStreamStreamer{}, errUnknownPhysicalValue
	}

	body.State = WalStreamStreamerState(streamer.State)

	return body, nil
}
