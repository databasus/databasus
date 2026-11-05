package streamhealth

import (
	"slices"
	"time"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// GetVerdict decides the one health level of a WAL-streaming database and lists
// the reasons behind it, broken ones first, and the notices that never change
// it. The first matching level wins: not streaming, collecting data, broken,
// needs attention, healthy.
func GetVerdict(input VerdictInput) Verdict {
	if !input.IsStreamingExpected {
		return Verdict{Level: HealthLevelNotStreaming, Reasons: []Reason{}, Notices: []Notice{}}
	}

	if isCollectingData(input) {
		return Verdict{Level: HealthLevelCollectingData, Reasons: []Reason{}, Notices: []Notice{}}
	}

	verdictSamples := newVerdictSamples(input)

	reasons := collectReasons(input, verdictSamples)

	slices.SortStableFunc(reasons, func(first, second Reason) int {
		return getReasonLevelRank(first.Level) - getReasonLevelRank(second.Level)
	})

	return Verdict{
		Level:   getHealthLevel(reasons),
		Reasons: reasons,
		Notices: collectNotices(input, verdictSamples),
	}
}

// The newest sample and the one before it, as far as the verdict trusts them.
// A sample older than staleSampleAge no longer describes now, and a sample
// without source-side values says nothing about the slot or the archive.
type verdictSamples struct {
	newest   *catalog.WalStreamSample
	previous *catalog.WalStreamSample

	isNewestFresh            bool
	isNewestWithSourceValues bool
}

func newVerdictSamples(input VerdictInput) verdictSamples {
	newest := input.NewestSample
	if newest == nil {
		return verdictSamples{}
	}

	isNewestFresh := input.Now.Sub(newest.SampledAt) <= staleSampleAge

	var previous *catalog.WalStreamSample

	sampleCount := len(input.RecentSamples)
	if sampleCount >= 2 && input.RecentSamples[sampleCount-1].ID == newest.ID {
		previous = input.RecentSamples[sampleCount-2]
	}

	return verdictSamples{
		newest:                   newest,
		previous:                 previous,
		isNewestFresh:            isNewestFresh,
		isNewestWithSourceValues: isNewestFresh && newest.IsSourceReachable(),
	}
}

// A single unusual sample must not change the verdict, so a condition read
// from samples counts only when it also held in the sample before.
func (s verdictSamples) isTrueForNewestTwo(
	condition func(sample *catalog.WalStreamSample) bool,
) bool {
	return s.previous != nil && condition(s.newest) && condition(s.previous)
}

// Right after a start or a restart no sample of the session exists yet, which
// is not a failure. A streamer that has already been failing for a while is
// not collecting anything, however young its current session is.
func isCollectingData(input VerdictInput) bool {
	streamer := input.Streamer
	if streamer == nil {
		return true
	}

	if isStreamerFailingLongEnough(streamer, input.Now) {
		return false
	}

	hasSampleInSession := input.NewestSample != nil && input.NewestSample.SampledAt.After(streamer.StartedAt)

	return !hasSampleInSession && input.Now.Sub(streamer.StartedAt) < collectingDataPeriod
}

func isStreamerFailingLongEnough(streamer *catalog.WalStreamer, now time.Time) bool {
	return streamer.FailingSince != nil && now.Sub(*streamer.FailingSince) >= streamingStoppedAfter
}

func getHealthLevel(reasons []Reason) HealthLevel {
	if len(reasons) == 0 {
		return HealthLevelHealthy
	}

	if reasons[0].Level == ReasonLevelBroken {
		return HealthLevelBroken
	}

	return HealthLevelNeedsAttention
}

func getReasonLevelRank(level ReasonLevel) int {
	switch level {
	case ReasonLevelBroken:
		return 0
	case ReasonLevelNeedsAttention:
		return 1
	case ReasonLevelInfo:
		return 2
	default:
		return 2
	}
}

func collectReasons(input VerdictInput, samples verdictSamples) []Reason {
	reasonGetters := []func(VerdictInput, verdictSamples) *Reason{
		getStreamingStoppedReason,
		getSourceUnreachableReason,
		getWalChainBrokenReason,
		getSlotMissingReason,
		getSlotLostReason,
		getArchiveNotAdvancingReason,
		getArchiveSlowerThanSourceReason,
		getSlotKeepsExtraWalReason,
		getSlotLimitApproachingReason,
		getNearSlotRebuildReason,
		getLocalQueueFillingReason,
		getFrequentRestartsReason,
		getForcedRotationRefusedReason,
	}

	reasons := make([]Reason, 0)

	for _, getReason := range reasonGetters {
		if reason := getReason(input, samples); reason != nil {
			reasons = append(reasons, *reason)
		}
	}

	return reasons
}

// An unreachable source is its own reason, and it explains why streaming
// stopped, so it is not reported twice.
func getStreamingStoppedReason(input VerdictInput, samples verdictSamples) *Reason {
	newest := samples.newest

	if newest != nil &&
		newest.SourceState == catalog.WalStreamSourceStateUnreachable {
		return nil
	}

	if newest != nil && !samples.isNewestFresh {
		return &Reason{Code: ReasonCodeStreamingStopped, Level: ReasonLevelBroken, Since: new(newest.SampledAt)}
	}

	streamer := input.Streamer
	if streamer == nil {
		return nil
	}

	if isStreamerFailingLongEnough(streamer, input.Now) {
		return &Reason{Code: ReasonCodeStreamingStopped, Level: ReasonLevelBroken, Since: new(*streamer.FailingSince)}
	}

	if newest == nil {
		return &Reason{Code: ReasonCodeStreamingStopped, Level: ReasonLevelBroken, Since: new(streamer.StartedAt)}
	}

	return nil
}

// Read from the newest sample even when streaming stopped after it: a source
// that went down is what stopped streaming.
func getSourceUnreachableReason(input VerdictInput, samples verdictSamples) *Reason {
	newest := samples.newest

	if newest == nil ||
		newest.SourceState != catalog.WalStreamSourceStateUnreachable ||
		newest.SourceUnreachableSince == nil {
		return nil
	}

	unreachableFor := input.Now.Sub(*newest.SourceUnreachableSince)

	switch {
	case unreachableFor >= sourceUnreachableBrokenAfter:
		return &Reason{
			Code:  ReasonCodeSourceUnreachable,
			Level: ReasonLevelBroken,
			Since: new(*newest.SourceUnreachableSince),
		}

	case unreachableFor >= sourceUnreachableAttentionAfter:
		return &Reason{
			Code:  ReasonCodeSourceUnreachable,
			Level: ReasonLevelNeedsAttention,
			Since: new(*newest.SourceUnreachableSince),
		}

	default:
		return nil
	}
}

// Reads the chain, not the slot: a rebuilt slot looks healthy within one tick
// while the chain stays broken until a newer full backup starts a new one. A
// gap counts only once the archive progress point has passed it, because that
// point never passes a pending segment, so a hole left by out-of-order parallel
// uploads is still above it.
func getWalChainBrokenReason(input VerdictInput, samples verdictSamples) *Reason {
	newest := samples.newest
	if newest == nil || newest.ArchiveProgressLSN == nil {
		return nil
	}

	for _, gap := range input.RestoreBlockingGaps {
		if gap.End <= *newest.ArchiveProgressLSN {
			return &Reason{Code: ReasonCodeWalChainBroken, Level: ReasonLevelBroken}
		}
	}

	return nil
}

func getSlotMissingReason(_ VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestFresh ||
		samples.newest.SourceState != catalog.WalStreamSourceStateSlotMissing {
		return nil
	}

	return &Reason{Code: ReasonCodeSlotMissing, Level: ReasonLevelBroken}
}

func getSlotLostReason(_ VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestWithSourceValues || !hasSlotWalStatus(samples.newest, slotWalStatusLost) {
		return nil
	}

	return &Reason{Code: ReasonCodeSlotLost, Level: ReasonLevelBroken}
}

// The broken level fires together with the "archiving fell behind"
// notification, from the same tracker and the same threshold.
func getArchiveNotAdvancingReason(input VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestWithSourceValues || samples.newest.ArchiveStalledSince == nil {
		return nil
	}

	stalledSince := *samples.newest.ArchiveStalledSince
	stalledFor := input.Now.Sub(stalledSince)

	switch {
	case stalledFor >= DefaultArchiveStalenessThreshold:
		return &Reason{Code: ReasonCodeArchiveNotAdvancing, Level: ReasonLevelBroken, Since: new(stalledSince)}

	case stalledFor >= archiveNotAdvancingAttentionAfter:
		return &Reason{Code: ReasonCodeArchiveNotAdvancing, Level: ReasonLevelNeedsAttention, Since: new(stalledSince)}

	default:
		return nil
	}
}

func getArchiveSlowerThanSourceReason(input VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestWithSourceValues {
		return nil
	}

	thresholdBytes := getBacklogThresholdBytes(samples.newest)

	isFallingBehind := samples.isTrueForNewestTwo(func(sample *catalog.WalStreamSample) bool {
		if !sample.IsSourceReachable() {
			return false
		}

		growthBytes, isKnown := getUnarchivedWalGrowthBytes(input.RecentSamples, sample.SampledAt)

		return isKnown && growthBytes >= thresholdBytes
	})
	if !isFallingBehind {
		return nil
	}

	growthBytes, _ := getUnarchivedWalGrowthBytes(input.RecentSamples, samples.newest.SampledAt)

	return &Reason{
		Code:           ReasonCodeArchiveSlowerThanSource,
		Level:          ReasonLevelNeedsAttention,
		ValueBytes:     new(growthBytes),
		ThresholdBytes: new(thresholdBytes),
	}
}

// An unreserved slot is about to lose WAL at the next checkpoint, and an
// extended one is only a problem while it keeps growing: on a busy cluster a
// steadily extended slot is routine and shows as a notice instead.
func getSlotKeepsExtraWalReason(input VerdictInput, samples verdictSamples) *Reason {
	if !isSlotKeepingExtraWalLongEnough(input, samples) {
		return nil
	}

	newest := samples.newest

	if !hasSlotWalStatus(newest, slotWalStatusUnreserved) && !isSlotLagGrowing(input, newest) {
		return nil
	}

	return &Reason{
		Code:  ReasonCodeSlotKeepsExtraWal,
		Level: ReasonLevelNeedsAttention,
		Since: new(*newest.SlotExtraWalSince),
	}
}

func getSlotLimitApproachingReason(input VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestWithSourceValues {
		return nil
	}

	isApproaching := samples.isTrueForNewestTwo(func(sample *catalog.WalStreamSample) bool {
		estimatedSeconds, isKnown := getEstimatedSlotLimitSeconds(input.RecentSamples, sample)

		return isKnown && estimatedSeconds < int64(slotLimitApproachingHorizon.Seconds())
	})
	if !isApproaching {
		return nil
	}

	estimatedSeconds, _ := getEstimatedSlotLimitSeconds(input.RecentSamples, samples.newest)

	return &Reason{
		Code:             ReasonCodeSlotLimitApproaching,
		Level:            ReasonLevelNeedsAttention,
		ValueBytes:       new(*samples.newest.SlotSafeWalSizeBytes),
		EstimatedSeconds: new(estimatedSeconds),
	}
}

// Stays listed while the receiver is paused: once it runs again above the
// rebuild lag, the slot is rebuilt if the running receiver does not reduce the
// lag, so a paused receiver above it is still at risk.
func getNearSlotRebuildReason(input VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestWithSourceValues || input.SlotRebuildLagBytes <= 0 {
		return nil
	}

	isNearRebuild := samples.isTrueForNewestTwo(func(sample *catalog.WalStreamSample) bool {
		return sample.SlotLagBytes != nil &&
			*sample.SlotLagBytes*100 >= input.SlotRebuildLagBytes*nearSlotRebuildPercent &&
			!hasSlotWalStatus(sample, slotWalStatusExtended) &&
			!hasSlotWalStatus(sample, slotWalStatusUnreserved)
	})
	if !isNearRebuild {
		return nil
	}

	return &Reason{
		Code:           ReasonCodeNearSlotRebuild,
		Level:          ReasonLevelNeedsAttention,
		ValueBytes:     new(*samples.newest.SlotLagBytes),
		ThresholdBytes: new(input.SlotRebuildLagBytes),
	}
}

func getLocalQueueFillingReason(_ VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestFresh {
		return nil
	}

	isFilling := samples.isTrueForNewestTwo(func(sample *catalog.WalStreamSample) bool {
		return sample.LocalQueuePauseBytes > 0 &&
			sample.LocalQueueBytes*100 >= sample.LocalQueuePauseBytes*localQueueFillingPercent
	})
	if !isFilling {
		return nil
	}

	return &Reason{
		Code:           ReasonCodeLocalQueueFilling,
		Level:          ReasonLevelNeedsAttention,
		ValueBytes:     new(samples.newest.LocalQueueBytes),
		ThresholdBytes: new(samples.newest.LocalQueuePauseBytes),
	}
}

func getFrequentRestartsReason(input VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestFresh {
		return nil
	}

	restarts := GetRestartsLastHour(input.RecentSamples, input.SessionStartsLastHour, input.Now)
	if restarts < frequentRestartsCount {
		return nil
	}

	return &Reason{Code: ReasonCodeFrequentRestarts, Level: ReasonLevelNeedsAttention, Count: new(restarts)}
}

func getForcedRotationRefusedReason(_ VerdictInput, samples verdictSamples) *Reason {
	if !samples.isNewestFresh || !samples.newest.IsForcedRotationDenied {
		return nil
	}

	return &Reason{Code: ReasonCodeForcedRotationRefused, Level: ReasonLevelNeedsAttention}
}

// PostgreSQL keeps WAL for a slot without a limit by default, and a steadily
// extended slot is routine on a busy cluster, so neither changes the level:
// shown as needs attention, they would stay orange for good and teach the
// reader to ignore orange.
func collectNotices(input VerdictInput, samples verdictSamples) []Notice {
	notices := make([]Notice, 0)

	if samples.isNewestWithSourceValues &&
		samples.newest.SlotSafeWalSizeBytes == nil &&
		!hasSlotWalStatus(samples.newest, slotWalStatusLost) {
		notices = append(notices, Notice{Code: NoticeCodeNoSlotWalLimit, Level: ReasonLevelInfo})
	}

	if isSlotKeepingExtraWalLongEnough(input, samples) &&
		hasSlotWalStatus(samples.newest, slotWalStatusExtended) &&
		!isSlotLagGrowing(input, samples.newest) {
		notices = append(notices, Notice{
			Code:  NoticeCodeSlotKeepsExtraWalSteadily,
			Level: ReasonLevelInfo,
			Since: new(*samples.newest.SlotExtraWalSince),
		})
	}

	if !input.HasCompletedFullBackup {
		notices = append(notices, Notice{Code: NoticeCodeNoFullBackupYet, Level: ReasonLevelInfo})
	}

	return notices
}

func isSlotKeepingExtraWalLongEnough(input VerdictInput, samples verdictSamples) bool {
	return samples.isNewestWithSourceValues &&
		samples.newest.SlotExtraWalSince != nil &&
		input.Now.Sub(*samples.newest.SlotExtraWalSince) > WarningSlotStatusHoldPeriod
}

// Growth below the backlog threshold is the same upload noise the unarchived
// WAL shows, so it does not count as growing.
func isSlotLagGrowing(input VerdictInput, newest *catalog.WalStreamSample) bool {
	changeBytesPerSecond, isKnown := getChangeBytesPerSecond(input.RecentSamples, newest.SampledAt, readSlotLagBytes)

	return isKnown && getGrowthOverWindowBytes(changeBytesPerSecond) >= getBacklogThresholdBytes(newest)
}

func getUnarchivedWalGrowthBytes(
	samples []*catalog.WalStreamSample,
	windowEnd time.Time,
) (growthBytes int64, isKnown bool) {
	changeBytesPerSecond, isKnown := getChangeBytesPerSecond(samples, windowEnd, readUnarchivedWalBytes)
	if !isKnown {
		return 0, false
	}

	return getGrowthOverWindowBytes(changeBytesPerSecond), true
}

func getEstimatedSlotLimitSeconds(
	samples []*catalog.WalStreamSample,
	sample *catalog.WalStreamSample,
) (estimatedSeconds int64, isKnown bool) {
	if !sample.IsSourceReachable() || sample.SlotSafeWalSizeBytes == nil {
		return 0, false
	}

	changeBytesPerSecond, isKnown := getChangeBytesPerSecond(samples, sample.SampledAt, readSlotLagBytes)
	if !isKnown || changeBytesPerSecond <= 0 {
		return 0, false
	}

	return int64(float64(*sample.SlotSafeWalSizeBytes) / changeBytesPerSecond), true
}

func hasSlotWalStatus(sample *catalog.WalStreamSample, walStatus string) bool {
	return sample.SlotWalStatus != nil && *sample.SlotWalStatus == walStatus
}
