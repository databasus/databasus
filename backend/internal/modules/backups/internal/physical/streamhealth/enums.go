package streamhealth

type HealthLevel string

const (
	HealthLevelNotStreaming   HealthLevel = "NOT_STREAMING"
	HealthLevelCollectingData HealthLevel = "COLLECTING_DATA"
	HealthLevelBroken         HealthLevel = "BROKEN"
	HealthLevelNeedsAttention HealthLevel = "NEEDS_ATTENTION"
	HealthLevelHealthy        HealthLevel = "HEALTHY"
)

type ReasonLevel string

const (
	ReasonLevelBroken         ReasonLevel = "BROKEN"
	ReasonLevelNeedsAttention ReasonLevel = "NEEDS_ATTENTION"
	ReasonLevelInfo           ReasonLevel = "INFO"
)

type ReasonCode string

const (
	ReasonCodeStreamingStopped        ReasonCode = "STREAMING_STOPPED"
	ReasonCodeSourceUnreachable       ReasonCode = "SOURCE_UNREACHABLE"
	ReasonCodeWalChainBroken          ReasonCode = "WAL_CHAIN_BROKEN"
	ReasonCodeSlotMissing             ReasonCode = "SLOT_MISSING"
	ReasonCodeSlotLost                ReasonCode = "SLOT_LOST"
	ReasonCodeArchiveNotAdvancing     ReasonCode = "ARCHIVE_NOT_ADVANCING"
	ReasonCodeArchiveSlowerThanSource ReasonCode = "ARCHIVE_SLOWER_THAN_SOURCE"
	ReasonCodeSlotKeepsExtraWal       ReasonCode = "SLOT_KEEPS_EXTRA_WAL"
	ReasonCodeSlotLimitApproaching    ReasonCode = "SLOT_LIMIT_APPROACHING"
	ReasonCodeNearSlotRebuild         ReasonCode = "NEAR_SLOT_REBUILD"
	ReasonCodeLocalQueueFilling       ReasonCode = "LOCAL_QUEUE_FILLING"
	ReasonCodeFrequentRestarts        ReasonCode = "FREQUENT_RESTARTS"
	ReasonCodeForcedRotationRefused   ReasonCode = "FORCED_ROTATION_REFUSED"
)

type NoticeCode string

const (
	NoticeCodeNoSlotWalLimit            NoticeCode = "NO_SLOT_WAL_LIMIT"
	NoticeCodeSlotKeepsExtraWalSteadily NoticeCode = "SLOT_KEEPS_EXTRA_WAL_STEADILY"
	NoticeCodeNoFullBackupYet           NoticeCode = "NO_FULL_BACKUP_YET"
)

type StreamerState string

const (
	StreamerStateRunning    StreamerState = "RUNNING"
	StreamerStateNotRunning StreamerState = "NOT_RUNNING"
	StreamerStateNotStarted StreamerState = "NOT_STARTED"
)

type KeepingUpState string

const (
	KeepingUpStateUnknown       KeepingUpState = "UNKNOWN"
	KeepingUpStateFallingBehind KeepingUpState = "FALLING_BEHIND"
	KeepingUpStateCatchingUp    KeepingUpState = "CATCHING_UP"
	KeepingUpStateIdle          KeepingUpState = "IDLE"
	KeepingUpStateKeepingUp     KeepingUpState = "KEEPING_UP"
)

// Values of the source's pg_replication_slots.wal_status that the verdict reads.
const (
	slotWalStatusExtended   = "extended"
	slotWalStatusUnreserved = "unreserved"
	slotWalStatusLost       = "lost"
)
