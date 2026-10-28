package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamLive(live physical.GetWalStreamLiveResponse) (WalStreamLive, error) {
	body := WalStreamLive{}

	if live.Current != nil {
		converted, err := toWalStreamLiveValues(*live.Current)
		if err != nil {
			return WalStreamLive{}, err
		}

		body.Current = &converted
	}

	body.IsStreamerRunning = live.IsStreamerRunning

	return body, nil
}
