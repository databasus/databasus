package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

func toWalStreamNotice(notice streamhealth.Notice) (WalStreamNotice, error) {
	body := WalStreamNotice{}

	switch string(notice.Code) {
	case "NO_SLOT_WAL_LIMIT", "SLOT_KEEPS_EXTRA_WAL_STEADILY", "NO_FULL_BACKUP_YET":
	default:
		return WalStreamNotice{}, errUnknownPhysicalValue
	}

	body.Code = WalStreamNoticeCode(notice.Code)

	switch string(notice.Level) {
	case "BROKEN", "NEEDS_ATTENTION", "INFO":
	default:
		return WalStreamNotice{}, errUnknownPhysicalValue
	}

	body.Level = WalStreamReasonLevel(notice.Level)
	body.Since = notice.Since

	return body, nil
}
