package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamTimeRange(timeRange physical.WalStreamTimeRange) (WalStreamTimeRange, error) {
	body := WalStreamTimeRange{}
	body.From = timeRange.From
	body.To = timeRange.To

	return body, nil
}
