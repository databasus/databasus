package catalog

type BackupStatus string

const (
	// pg_basebackup spawned (or about to spawn), not yet finished.
	BackupStatusInProgress BackupStatus = "IN_PROGRESS"

	// Artifact in storage, all metadata populated. Chain is extendable
	// from this row.
	BackupStatusCompleted BackupStatus = "COMPLETED"

	// Transient failure (network blip, pg_basebackup spurious error,
	// storage upload retry exhausted, byte-stall watcher tripped). Chain is
	// still valid for this attempt's parent; scheduler retries the SAME kind
	// (FULL or INCR) on next tick per retry policy.
	BackupStatusError BackupStatus = "ERROR"

	// Chain cannot be extended further from this point. Next attempt MUST
	// escalate to a fresh FULL (new chain identity). Rare for FULL but
	// possible (corrupted manifest mid-stream, sys_id mismatch caught
	// partway, start_lsn outside any known timeline range). Common for INCR
	// (summaries expired, summarizer off, parent manifest missing).
	BackupStatusChainBroken BackupStatus = "CHAIN_BROKEN"

	// User- or system-initiated cancel. See BackupErrorReason for
	// the cancel variants.
	BackupStatusCanceled BackupStatus = "CANCELED"
)

type BackupErrorReason string

const (
	BackupErrorPgBasebackupFailed       BackupErrorReason = "PG_BASEBACKUP_FAILED"
	BackupErrorStorageUploadFailed      BackupErrorReason = "STORAGE_UPLOAD_FAILED"
	BackupErrorNetworkFailure           BackupErrorReason = "NETWORK_FAILURE"
	BackupErrorNetworkStallTimeout      BackupErrorReason = "NETWORK_STALL_TIMEOUT"
	BackupErrorApplicationRestart       BackupErrorReason = "APPLICATION_RESTART"
	BackupErrorSystemIdentifierMismatch BackupErrorReason = "SYSTEM_IDENTIFIER_MISMATCH"
	BackupErrorTimelineRegression       BackupErrorReason = "TIMELINE_REGRESSION"
	BackupErrorTimelineSwitchDetected   BackupErrorReason = "TIMELINE_SWITCH_DETECTED"
	BackupErrorFailoverDuringBackup     BackupErrorReason = "FAILOVER_DURING_BACKUP"
	BackupErrorManifestCorrupted        BackupErrorReason = "MANIFEST_CORRUPTED"
	BackupErrorStartLsnOutsideTimeline  BackupErrorReason = "START_LSN_OUTSIDE_TIMELINE_RANGE"

	// Covers both an explicit user cancel on a running backup AND
	// running backups cancelled when the user disables backups.
	BackupErrorCanceledByUser BackupErrorReason = "CANCELED_BY_USER"

	// INCR-specific and chain-killing (status = CHAIN_BROKEN).
	BackupErrorSummariesExpired      BackupErrorReason = "SUMMARIES_EXPIRED"
	BackupErrorSummarizerOff         BackupErrorReason = "SUMMARIZER_OFF"
	BackupErrorParentManifestMissing BackupErrorReason = "PARENT_MANIFEST_MISSING"

	// INCR-specific and transient (status = ERROR): the summarizer is on and
	// covers the parent, but its process is not publishing summaries. The chain
	// stays extendable and the next cadence tick retries it; only a run of
	// consecutive failures closes the chain.
	BackupErrorSummarizerFallingBehind BackupErrorReason = "SUMMARIZER_FALLING_BEHIND"
)

type WalStreamerStatus string

const (
	WalStreamerStatusRunning WalStreamerStatus = "RUNNING"
	WalStreamerStatusFailed  WalStreamerStatus = "FAILED"
)

type BackupCompression string

const (
	BackupCompressionZstd BackupCompression = "ZSTD"
	BackupCompressionGzip BackupCompression = "GZIP"
	BackupCompressionNone BackupCompression = "NONE"
)

type BackupType string

const (
	BackupTypeFull        BackupType = "FULL"
	BackupTypeIncremental BackupType = "INCREMENTAL"

	// BackupTypeWal tags a committed WAL segment in the flat backup
	// list. WAL has no dedicated backup table-row identity beyond the segment,
	// so this value exists only as a list/UI discriminator, never as a backup
	// the scheduler creates.
	BackupTypeWal BackupType = "WAL"
)

type WalStreamSourceState string

const (
	WalStreamSourceStateReachable   WalStreamSourceState = "REACHABLE"
	WalStreamSourceStateUnreachable WalStreamSourceState = "UNREACHABLE"

	// The source answered, but Databasus's replication slot no longer exists there.
	// Unlike an unreachable source, this means the WAL chain is already broken.
	WalStreamSourceStateSlotMissing WalStreamSourceState = "SLOT_MISSING"
)
