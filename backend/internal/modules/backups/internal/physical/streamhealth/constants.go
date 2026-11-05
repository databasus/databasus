package streamhealth

import "time"

const (
	// The WAL streamer's lag monitor probes the source and records one sample on
	// this cadence. It balances detection latency against query load on the
	// source, and every rate divides by it: one sample stands for this much time.
	SampleInterval = 30 * time.Second

	SampleRetention = 7 * 24 * time.Hour

	// A warning wal_status must persist this long before the WAL streamer alerts
	// on it, so a transient write burst stays quiet. A slot lag above the
	// operator's threshold must go unreduced for this much receiver running time
	// before the streamer rebuilds the slot, so a receiver catching up keeps the
	// chain.
	WarningSlotStatusHoldPeriod = 5 * time.Minute

	// Over one hold period of running time, a lag above the threshold that shrinks
	// by less than this share of the threshold counts as not reduced. It keeps a
	// lag that wobbles on a busy source from passing as a drain.
	SlotLagMinDrainPercentOfThreshold = 5

	// A rarely-written database can legitimately hold its newest WAL locally for a
	// while, but past this the recovery point has silently stopped tracking the
	// source and only the operator can decide what to do about it.
	DefaultArchiveStalenessThreshold = 30 * time.Minute

	// A streamer row whose heartbeat is older than this is assumed dead and
	// reclaimable by another process. Heartbeats fire every manager pass
	// (15 s), so 90 s = 6 missed beats — tolerant of GC pauses without leaving a
	// dead streamer unclaimed for long.
	StreamerHeartbeatStaleness = 90 * time.Second
)

// Verdict thresholds. Each sits just ahead of something that already acts, so
// the screen warns before Databasus pauses, rebuilds or notifies.
const (
	// Four missed samples: past this the newest sample no longer describes now.
	staleSampleAge = 2 * time.Minute

	// A session younger than this without a sample is still starting up.
	collectingDataPeriod = 2 * time.Minute

	streamingStoppedAfter = 1 * time.Minute

	sourceUnreachableAttentionAfter = 1 * time.Minute
	sourceUnreachableBrokenAfter    = 5 * time.Minute

	archiveNotAdvancingAttentionAfter = 10 * time.Minute

	// A 30-second rate swings with every upload batch, while an hour hides a
	// backlog that started five minutes ago.
	keepingUpWindow = 10 * time.Minute

	// The lowest value at each end of the window strips the sawtooth of in-flight
	// uploads, which reaches four segments plus the open one in normal operation.
	keepingUpWindowEdge = 150 * time.Second

	minBacklogThresholdBytes int64 = 64 * 1024 * 1024
	backlogThresholdSegments int64 = 4

	slotLimitApproachingHorizon = 1 * time.Hour

	// Percent of the configured rebuild lag: warns before Databasus sacrifices the
	// chain by rebuilding the slot.
	nearSlotRebuildPercent int64 = 80

	// Percent of the pause size: warns before back pressure stops the receiver.
	localQueueFillingPercent int64 = 50

	// Tolerates a deploy and one network blip an hour.
	frequentRestartsCount  = 3
	frequentRestartsWindow = 1 * time.Hour

	idleSourceWrittenBytes int64 = 1024 * 1024
)
