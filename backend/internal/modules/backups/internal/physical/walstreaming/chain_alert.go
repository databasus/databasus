package walstreaming

import "github.com/google/uuid"

type chainAlertKind string

const (
	chainAlertStreamerFailed chainAlertKind = "STREAMER_FAILED"
	chainAlertSlotRebuilt    chainAlertKind = "SLOT_REBUILT"
	chainAlertChainAtRisk    chainAlertKind = "CHAIN_AT_RISK"
	chainAlertArchiveStale   chainAlertKind = "WAL_ARCHIVE_STALE"
	chainAlertRotationDenied chainAlertKind = "WAL_ROTATION_DENIED"
)

type chainAlertKey struct {
	DatabaseID uuid.UUID
	Kind       chainAlertKind
}

type chainAlert struct {
	Kind    chainAlertKind
	Heading string
	Message string
}
