package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
)

func toWalStreamHealth(health physical.WalStreamHealth) (WalStreamHealth, error) {
	body := WalStreamHealth{}
	body.IsFullBackupPending = health.IsFullBackupPending

	switch string(health.Level) {
	case "NOT_STREAMING", "COLLECTING_DATA", "BROKEN", "NEEDS_ATTENTION", "HEALTHY":
	default:
		return WalStreamHealth{}, errUnknownPhysicalValue
	}

	body.Level = WalStreamHealthLevel(health.Level)
	body.Notices = make([]WalStreamNotice, 0, len(health.Notices))

	for _, notice := range health.Notices {
		converted, err := toWalStreamNotice(notice)
		if err != nil {
			return WalStreamHealth{}, err
		}

		body.Notices = append(body.Notices, converted)
	}

	body.Reasons = make([]WalStreamReason, 0, len(health.Reasons))

	for _, reason := range health.Reasons {
		converted, err := toWalStreamReason(reason)
		if err != nil {
			return WalStreamHealth{}, err
		}

		body.Reasons = append(body.Reasons, converted)
	}

	return body, nil
}
