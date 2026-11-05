package streamhealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type verdictTestCase struct {
	name            string
	buildInput      func() VerdictInput
	expectedLevel   HealthLevel
	expectedReasons []reasonExpectation
	expectedNotices []NoticeCode
}

func runVerdictTestCases(t *testing.T, testCases []verdictTestCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			verdict := GetVerdict(testCase.buildInput())

			assert.Equal(t, testCase.expectedLevel, verdict.Level)
			assert.Equal(t, append([]reasonExpectation{}, testCase.expectedReasons...),
				getReasonExpectations(verdict.Reasons))
			assert.Equal(t, append([]NoticeCode{}, testCase.expectedNotices...), getNoticeCodes(verdict.Notices))
		})
	}
}

func inputWithNewestTwo(modify func(sample *catalog.WalStreamSample)) func() VerdictInput {
	return func() VerdictInput {
		return newTestVerdictInput(buildTestSamples(0, modifyNewestTwo(modify)))
	}
}

func inputWithEverySample(
	modify func(index int, sample *catalog.WalStreamSample),
) func() VerdictInput {
	return func() VerdictInput {
		return newTestVerdictInput(buildTestSamples(0, modify))
	}
}

func unreachableSince(unreachableFor time.Duration) func(sample *catalog.WalStreamSample) {
	return func(sample *catalog.WalStreamSample) {
		makeUnreachable(sample, getTestNow().Add(-unreachableFor))
	}
}

func makeUnreachable(sample *catalog.WalStreamSample, since time.Time) {
	sample.SourceState = catalog.WalStreamSourceStateUnreachable
	sample.SourceUnreachableSince = new(since)
	sample.SourceCurrentLSN = nil
	sample.SourceWrittenBytes = nil
	sample.SlotWalStatus = nil
	sample.SlotLagBytes = nil
	sample.SlotSafeWalSizeBytes = nil
	sample.UnarchivedWalBytes = nil
}

func archiveStalledFor(stalledFor time.Duration) func(sample *catalog.WalStreamSample) {
	return func(sample *catalog.WalStreamSample) {
		sample.ArchiveStalledSince = new(getTestNow().Add(-stalledFor))
	}
}

func slotStatusSince(walStatus string, heldFor time.Duration) func(sample *catalog.WalStreamSample) {
	return func(sample *catalog.WalStreamSample) {
		sample.SlotWalStatus = new(walStatus)
		sample.SlotExtraWalSince = new(getTestNow().Add(-heldFor))
	}
}

// Each sample adds stepBytes to the value, so the value grows by twenty steps
// over the keeping-up window.
func growingUnarchivedWal(stepBytes int64) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		sample.UnarchivedWalBytes = new(8*testMiB + int64(index)*stepBytes)
	}
}

func growingSlotLag(stepBytes int64) func(index int, sample *catalog.WalStreamSample) {
	return func(index int, sample *catalog.WalStreamSample) {
		sample.SlotLagBytes = new(2*testMiB + int64(index)*stepBytes)
	}
}

func Test_GetVerdict_WithSpecScenarios_ReturnsLevelReasonsAndNotices(t *testing.T) {
	runVerdictTestCases(t, []verdictTestCase{
		{
			name:          "Everything in order",
			buildInput:    inputWithNewestTwo(func(*catalog.WalStreamSample) {}),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "Single unusual sample",
			buildInput: func() VerdictInput {
				return newTestVerdictInput(buildTestSamples(0, modifyNewestOnly(fillLocalQueue(testPauseBytes/2))))
			},
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:          "Short network blip",
			buildInput:    inputWithNewestTwo(unreachableSince(30 * time.Second)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "Source unreachable for a few minutes",
			buildInput:      inputWithNewestTwo(unreachableSince(3 * time.Minute)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSourceUnreachable, ReasonLevelNeedsAttention}},
		},
		{
			name: "Source down for a while",
			buildInput: func() VerdictInput {
				samples := buildTestSamples(4*time.Minute, modifyNewestTwo(unreachableSince(6*time.Minute)))
				input := newTestVerdictInput(samples)
				input.Streamer = newFailedTestStreamer(time.Minute, 3*time.Minute)

				return input
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeSourceUnreachable, ReasonLevelBroken}},
		},
		{
			name:            "Archive stuck for 12 minutes",
			buildInput:      inputWithNewestTwo(archiveStalledFor(12 * time.Minute)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveNotAdvancing, ReasonLevelNeedsAttention}},
		},
		{
			name:            "Archive stuck for 30 minutes",
			buildInput:      inputWithNewestTwo(archiveStalledFor(30 * time.Minute)),
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveNotAdvancing, ReasonLevelBroken}},
		},
		{
			name: "Streamer stopped",
			buildInput: func() VerdictInput {
				samples := buildTestSamples(
					10*time.Minute,
					modifyNewestTwo(func(sample *catalog.WalStreamSample) {
						sample.LocalQueueBytes = testPauseBytes
						sample.ArchiveStalledSince = new(getTestNow().Add(-time.Hour))
					}),
				)
				input := newTestVerdictInput(samples)
				input.Streamer = newFailedTestStreamer(10*time.Minute, 10*time.Minute)

				return input
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeStreamingStopped, ReasonLevelBroken}},
		},
		{
			name: "Restart after a deploy",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(10*time.Minute, nil))
				input.Streamer = newRunningTestStreamer()
				input.Streamer.StartedAt = getTestNow().Add(-20 * time.Second)
				input.Streamer.FailingSince = new(getTestNow().Add(-20 * time.Second))

				return input
			},
			expectedLevel: HealthLevelCollectingData,
		},
		{
			name: "Crash loop on a new database",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(nil)
				input.Streamer = newFailedTestStreamer(10*time.Second, 5*time.Minute)

				return input
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeStreamingStopped, ReasonLevelBroken}},
		},
		{
			name: "Slot removed on the source",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				makeUnreachable(sample, getTestNow())
				sample.SourceState = catalog.WalStreamSourceStateSlotMissing
				sample.SourceUnreachableSince = nil
			}),
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeSlotMissing, ReasonLevelBroken}},
		},
		{
			name: "Just enabled",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(nil)
				input.Streamer = nil

				return input
			},
			expectedLevel: HealthLevelCollectingData,
		},
		{
			name: "Backups disabled",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.IsStreamingExpected = false

				return input
			},
			expectedLevel: HealthLevelNotStreaming,
		},
		{
			name: "Several problems at once",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				makeUnreachable(sample, getTestNow())
				sample.SourceState = catalog.WalStreamSourceStateSlotMissing
				sample.SourceUnreachableSince = nil
				sample.LocalQueueBytes = testPauseBytes / 2
			}),
			expectedLevel: HealthLevelBroken,
			expectedReasons: []reasonExpectation{
				{ReasonCodeSlotMissing, ReasonLevelBroken},
				{ReasonCodeLocalQueueFilling, ReasonLevelNeedsAttention},
			},
		},
		{
			name: "Busy cluster with steady extra WAL",
			buildInput: inputWithEverySample(func(_ int, sample *catalog.WalStreamSample) {
				slotStatusSince("extended", time.Hour)(sample)
			}),
			expectedLevel:   HealthLevelHealthy,
			expectedNotices: []NoticeCode{NoticeCodeSlotKeepsExtraWalSteadily},
		},
		{
			name: "Paused receiver above the rebuild lag",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.IsReceiverRunning = false
				sample.SlotLagBytes = new(testRebuildLagBytes * 9 / 10)
			}),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeNearSlotRebuild, ReasonLevelNeedsAttention}},
		},
		{
			name: "No limit on the source",
			buildInput: inputWithEverySample(func(_ int, sample *catalog.WalStreamSample) {
				sample.SlotSafeWalSizeBytes = nil
			}),
			expectedLevel:   HealthLevelHealthy,
			expectedNotices: []NoticeCode{NoticeCodeNoSlotWalLimit},
		},
		{
			name: "Slot already lost",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.SlotWalStatus = new("lost")
				sample.SlotSafeWalSizeBytes = nil
			}),
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeSlotLost, ReasonLevelBroken}},
		},
		{
			name: "No full backup yet",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.HasCompletedFullBackup = false

				return input
			},
			expectedLevel:   HealthLevelHealthy,
			expectedNotices: []NoticeCode{NoticeCodeNoFullBackupYet},
		},
	})
}

func fillLocalQueue(queueBytes int64) func(sample *catalog.WalStreamSample) {
	return func(sample *catalog.WalStreamSample) {
		sample.LocalQueueBytes = queueBytes
	}
}

func Test_GetVerdict_AtReasonThresholds_ListsReasonOnlyPastThreshold(t *testing.T) {
	backlogStepAboveThreshold := (minBacklogThresholdBytes + 2*testMiB) / 20
	backlogStepBelowThreshold := (minBacklogThresholdBytes - 2*testMiB) / 20

	// The slot lag grows by stepBytes every 30 seconds, so the remaining allowance
	// runs out after allowance ÷ (stepBytes ÷ 30) seconds.
	slotLagStep := testMiB
	allowanceForSeconds := func(seconds int64) int64 { return seconds * slotLagStep / 30 }

	runVerdictTestCases(t, []verdictTestCase{
		{
			name:          "source unreachable just under a minute",
			buildInput:    inputWithNewestTwo(unreachableSince(59 * time.Second)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "source unreachable just over a minute",
			buildInput:      inputWithNewestTwo(unreachableSince(61 * time.Second)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSourceUnreachable, ReasonLevelNeedsAttention}},
		},
		{
			name:            "source unreachable just under five minutes",
			buildInput:      inputWithNewestTwo(unreachableSince(5*time.Minute - time.Second)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSourceUnreachable, ReasonLevelNeedsAttention}},
		},
		{
			name:            "source unreachable just over five minutes",
			buildInput:      inputWithNewestTwo(unreachableSince(5*time.Minute + time.Second)),
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeSourceUnreachable, ReasonLevelBroken}},
		},
		{
			name: "newest sample just under two minutes old",
			buildInput: func() VerdictInput {
				return newTestVerdictInput(buildTestSamples(2*time.Minute-time.Second, nil))
			},
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "newest sample just over two minutes old",
			buildInput: func() VerdictInput {
				return newTestVerdictInput(buildTestSamples(2*time.Minute+time.Second, nil))
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeStreamingStopped, ReasonLevelBroken}},
		},
		{
			name: "streamer failing just under a minute",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.Streamer = newFailedTestStreamer(time.Hour, time.Minute-time.Second)

				return input
			},
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "streamer failing just over a minute",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.Streamer = newFailedTestStreamer(time.Hour, time.Minute+time.Second)

				return input
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeStreamingStopped, ReasonLevelBroken}},
		},
		{
			name:          "archive stalled just under ten minutes",
			buildInput:    inputWithNewestTwo(archiveStalledFor(10*time.Minute - time.Second)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "archive stalled just over ten minutes",
			buildInput:      inputWithNewestTwo(archiveStalledFor(10*time.Minute + time.Second)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveNotAdvancing, ReasonLevelNeedsAttention}},
		},
		{
			name:            "archive stalled just under thirty minutes",
			buildInput:      inputWithNewestTwo(archiveStalledFor(30*time.Minute - time.Second)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveNotAdvancing, ReasonLevelNeedsAttention}},
		},
		{
			name:            "archive stalled just over thirty minutes",
			buildInput:      inputWithNewestTwo(archiveStalledFor(30*time.Minute + time.Second)),
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveNotAdvancing, ReasonLevelBroken}},
		},
		{
			name:          "unarchived wal growth just under the backlog threshold",
			buildInput:    inputWithEverySample(growingUnarchivedWal(backlogStepBelowThreshold)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "unarchived wal growth just over the backlog threshold",
			buildInput:      inputWithEverySample(growingUnarchivedWal(backlogStepAboveThreshold)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeArchiveSlowerThanSource, ReasonLevelNeedsAttention}},
		},
		{
			name: "unarchived wal sawtooth of four segments plus one",
			buildInput: inputWithEverySample(func(index int, sample *catalog.WalStreamSample) {
				sample.UnarchivedWalBytes = new(8*testMiB + int64(index%6)*testSegmentSizeBytes)
			}),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:          "slot unreserved just under five minutes",
			buildInput:    inputWithNewestTwo(slotStatusSince("unreserved", 5*time.Minute-time.Second)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "slot unreserved just over five minutes",
			buildInput:      inputWithNewestTwo(slotStatusSince("unreserved", 5*time.Minute+time.Second)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSlotKeepsExtraWal, ReasonLevelNeedsAttention}},
		},
		{
			name: "slot extended over five minutes while its lag grows",
			buildInput: inputWithEverySample(func(index int, sample *catalog.WalStreamSample) {
				growingSlotLag(backlogStepAboveThreshold)(index, sample)
				slotStatusSince("extended", 20*time.Minute)(sample)
			}),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSlotKeepsExtraWal, ReasonLevelNeedsAttention}},
		},
		{
			name: "slot allowance runs out just over an hour",
			buildInput: inputWithEverySample(func(index int, sample *catalog.WalStreamSample) {
				growingSlotLag(slotLagStep)(index, sample)
				sample.SlotSafeWalSizeBytes = new(allowanceForSeconds(3700))
			}),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "slot allowance runs out just under an hour",
			buildInput: inputWithEverySample(func(index int, sample *catalog.WalStreamSample) {
				growingSlotLag(slotLagStep)(index, sample)
				sample.SlotSafeWalSizeBytes = new(allowanceForSeconds(3500))
			}),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeSlotLimitApproaching, ReasonLevelNeedsAttention}},
		},
		{
			name: "slot lag just under 80% of the rebuild lag",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.SlotLagBytes = new(testRebuildLagBytes*8/10 - 1)
			}),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "slot lag just over 80% of the rebuild lag",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.SlotLagBytes = new(testRebuildLagBytes*8/10 + 1)
			}),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeNearSlotRebuild, ReasonLevelNeedsAttention}},
		},
		{
			name: "slot lag past the rebuild lag while extended",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.SlotLagBytes = new(testRebuildLagBytes * 2)
				sample.SlotWalStatus = new("extended")
			}),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:          "local queue just under half the pause size",
			buildInput:    inputWithNewestTwo(fillLocalQueue(testPauseBytes/2 - 1)),
			expectedLevel: HealthLevelHealthy,
		},
		{
			name:            "local queue at half the pause size",
			buildInput:      inputWithNewestTwo(fillLocalQueue(testPauseBytes / 2)),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeLocalQueueFilling, ReasonLevelNeedsAttention}},
		},
		{
			name: "two restarts in the last hour",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(
					buildTestSamples(0, modifyNewestOnly(func(sample *catalog.WalStreamSample) {
						sample.ReceiverRestarts = 1
					})),
				)
				input.SessionStartsLastHour = 1

				return input
			},
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "three restarts in the last hour",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(
					buildTestSamples(0, modifyNewestOnly(func(sample *catalog.WalStreamSample) {
						sample.ReceiverRestarts = 2
					})),
				)
				input.SessionStartsLastHour = 1

				return input
			},
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeFrequentRestarts, ReasonLevelNeedsAttention}},
		},
		{
			name: "forced rotation refused",
			buildInput: inputWithNewestTwo(func(sample *catalog.WalStreamSample) {
				sample.IsForcedRotationDenied = true
			}),
			expectedLevel:   HealthLevelNeedsAttention,
			expectedReasons: []reasonExpectation{{ReasonCodeForcedRotationRefused, ReasonLevelNeedsAttention}},
		},
	})
}

func Test_GetVerdict_WhenChainHasGap_ListsWalChainBrokenOnlyForGapBelowProgressPoint(t *testing.T) {
	gapBelowProgressPoint := catalog.LSNRange{
		Start: testArchiveProgressAt - walmath.LSN(2*testSegmentSizeBytes),
		End:   testArchiveProgressAt - walmath.LSN(testSegmentSizeBytes),
	}
	gapAboveProgressPoint := catalog.LSNRange{
		Start: testArchiveProgressAt,
		End:   testArchiveProgressAt + walmath.LSN(testSegmentSizeBytes),
	}

	runVerdictTestCases(t, []verdictTestCase{
		{
			name: "Slot rebuilt",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.RestoreBlockingGaps = []catalog.LSNRange{gapBelowProgressPoint}

				return input
			},
			expectedLevel:   HealthLevelBroken,
			expectedReasons: []reasonExpectation{{ReasonCodeWalChainBroken, ReasonLevelBroken}},
		},
		{
			name: "same chain without a gap",
			buildInput: func() VerdictInput {
				return newTestVerdictInput(buildTestSamples(0, nil))
			},
			expectedLevel: HealthLevelHealthy,
		},
		{
			name: "gap above the archive progress point",
			buildInput: func() VerdictInput {
				input := newTestVerdictInput(buildTestSamples(0, nil))
				input.RestoreBlockingGaps = []catalog.LSNRange{gapAboveProgressPoint}

				return input
			},
			expectedLevel: HealthLevelHealthy,
		},
	})
}

func Test_GetVerdict_WhenReasonsCarryValues_ReportsSinceAmountsAndThresholds(t *testing.T) {
	unreachableAt := getTestNow().Add(-3 * time.Minute)
	verdict := GetVerdict(inputWithNewestTwo(unreachableSince(3 * time.Minute))())

	sourceUnreachable := findReason(verdict.Reasons, ReasonCodeSourceUnreachable)
	require.NotNil(t, sourceUnreachable)
	require.NotNil(t, sourceUnreachable.Since)
	assert.Equal(t, unreachableAt, *sourceUnreachable.Since)

	newestSampleAt := getTestNow().Add(-10 * time.Minute)
	stoppedInput := newTestVerdictInput(buildTestSamples(10*time.Minute, nil))
	stoppedInput.Streamer = newFailedTestStreamer(10*time.Minute, 10*time.Minute)

	streamingStopped := findReason(GetVerdict(stoppedInput).Reasons, ReasonCodeStreamingStopped)
	require.NotNil(t, streamingStopped)
	require.NotNil(t, streamingStopped.Since)
	assert.Equal(t, newestSampleAt, *streamingStopped.Since)

	queueVerdict := GetVerdict(inputWithNewestTwo(fillLocalQueue(300 * testMiB))())

	localQueueFilling := findReason(queueVerdict.Reasons, ReasonCodeLocalQueueFilling)
	require.NotNil(t, localQueueFilling)
	assert.Equal(t, 300*testMiB, *localQueueFilling.ValueBytes)
	assert.Equal(t, testPauseBytes, *localQueueFilling.ThresholdBytes)
}

func Test_GetStreamerState_ForEachStreamerRow_ReportsRunningOnlyWithFreshHeartbeat(t *testing.T) {
	staleStreamer := newRunningTestStreamer()
	staleStreamer.LastHeartbeatAt = getTestNow().Add(-StreamerHeartbeatStaleness - time.Second)

	testCases := []struct {
		name          string
		streamer      *catalog.WalStreamer
		expectedState StreamerState
	}{
		{
			name:          "running with a fresh heartbeat",
			streamer:      newRunningTestStreamer(),
			expectedState: StreamerStateRunning,
		},
		{name: "Stale heartbeat", streamer: staleStreamer, expectedState: StreamerStateNotRunning},
		{
			name:          "failed",
			streamer:      newFailedTestStreamer(time.Minute, time.Minute),
			expectedState: StreamerStateNotRunning,
		},
		{name: "Streamer never started", streamer: nil, expectedState: StreamerStateNotStarted},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedState, GetStreamerState(testCase.streamer, getTestNow()))
		})
	}
}
