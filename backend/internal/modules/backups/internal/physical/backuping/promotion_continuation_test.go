package backuping

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

const (
	firstForkLSN  walmath.LSN = 0x3000148
	secondForkLSN walmath.LSN = 0x7000060
)

func TestDecideContinuationAcrossPromotion(t *testing.T) {
	t.Parallel()

	storageID := uuid.New()
	rootFull := newChainBackup(storageID, 1, 0x1000100)

	scenarios := []struct {
		name             string
		histories        []*catalog.WalHistoryFile
		newestBackup     *catalog.Backup
		liveTimelineID   int
		isChainContinued bool
	}{
		{
			name:             "continues a chain whose newest backup stopped before the fork point",
			histories:        []*catalog.WalHistoryFile{newPublishedFork(storageID, 2, 1, firstForkLSN)},
			newestBackup:     newChainBackup(storageID, 1, 0x2000100),
			liveTimelineID:   2,
			isChainContinued: true,
		},
		{
			name:             "continues a chain whose newest backup stopped exactly at the fork point",
			histories:        []*catalog.WalHistoryFile{newPublishedFork(storageID, 2, 1, firstForkLSN)},
			newestBackup:     newChainBackup(storageID, 1, firstForkLSN),
			liveTimelineID:   2,
			isChainContinued: true,
		},
		{
			name: "continues across two promotions",
			histories: []*catalog.WalHistoryFile{
				newPublishedFork(storageID, 2, 1, firstForkLSN),
				newPublishedFork(storageID, 3, 2, secondForkLSN),
			},
			newestBackup:     newChainBackup(storageID, 1, 0x2000100),
			liveTimelineID:   3,
			isChainContinued: true,
		},
		{
			name:           "breaks a chain whose newest backup stopped past the fork point",
			histories:      []*catalog.WalHistoryFile{newPublishedFork(storageID, 2, 1, firstForkLSN)},
			newestBackup:   newChainBackup(storageID, 1, 0x4000100),
			liveTimelineID: 2,
		},
		{
			name: "breaks a chain when the live timeline does not descend from the chain's timeline",
			histories: []*catalog.WalHistoryFile{
				newPublishedFork(storageID, 2, 1, firstForkLSN),
				newPublishedFork(storageID, 3, 1, secondForkLSN),
			},
			newestBackup:   newChainBackup(storageID, 2, 0x4000100),
			liveTimelineID: 3,
		},
		{
			name: "breaks a chain when the live timeline's history conflicts",
			histories: []*catalog.WalHistoryFile{
				withConflictingFork(newPublishedFork(storageID, 2, 1, firstForkLSN)),
			},
			newestBackup:   newChainBackup(storageID, 1, 0x2000100),
			liveTimelineID: 2,
		},
		{
			name:           "breaks a chain when the live timeline's history lies in another storage",
			histories:      []*catalog.WalHistoryFile{newPublishedFork(uuid.New(), 2, 1, firstForkLSN)},
			newestBackup:   newChainBackup(storageID, 1, 0x2000100),
			liveTimelineID: 2,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			detected := &timelines.TimelineDecision{
				Kind:               timelines.TimelineFailoverDetected,
				ExpectedTimelineID: scenario.newestBackup.TimelineID,
				LiveTimelineID:     scenario.liveTimelineID,
			}

			decision := decideContinuationAcrossPromotion(scenario.histories, chains.ChainExtension{
				RootFull: rootFull, NewestBackup: scenario.newestBackup, LiveTimelineID: scenario.liveTimelineID,
			}, detected)

			expectedKind := timelines.TimelineFailoverDetected
			if scenario.isChainContinued {
				expectedKind = timelines.TimelineContinueAcrossPromotion
			}
			require.Equal(t, expectedKind, decision.Kind)
			require.Equal(t, scenario.liveTimelineID, decision.LiveTimelineID)
		})
	}
}

func TestContinueAcrossPromotion(t *testing.T) {
	t.Parallel()

	storageID := uuid.New()
	extension := chains.ChainExtension{
		RootFull:       newChainBackup(storageID, 1, 0x1000100),
		NewestBackup:   newChainBackup(storageID, 1, 0x2000100),
		LiveTimelineID: 2,
	}
	detected := &timelines.TimelineDecision{
		Kind: timelines.TimelineFailoverDetected, ExpectedTimelineID: 1, LiveTimelineID: 2,
	}
	publishedFork := newPublishedFork(storageID, 2, 1, firstForkLSN)

	t.Run("breaks the chain when history publication fails", func(t *testing.T) {
		t.Parallel()
		hasLoadedHistories := false

		decision, err := continueAcrossPromotion(t.Context(), promotionContinuationSpec{
			Extension: extension,
			Decision:  detected,
			PublishHistory: func(context.Context, int) (*catalog.WalHistoryFile, error) {
				return nil, errors.New("storage unavailable")
			},
			LoadHistories: func(context.Context) ([]*catalog.WalHistoryFile, error) {
				hasLoadedHistories = true

				return []*catalog.WalHistoryFile{publishedFork}, nil
			},
			Logger: slog.New(slog.DiscardHandler),
		})

		require.NoError(t, err)
		require.Equal(t, timelines.TimelineFailoverDetected, decision.Kind)
		require.False(t, hasLoadedHistories, "a chain is never continued past a history that failed to publish")
		refusal, canProceed := timelineRefusalResult(decision, true)
		require.False(t, canProceed)
		require.Equal(t, catalog.BackupStatusChainBroken, refusal.Status)
		require.Equal(t, catalog.BackupErrorTimelineSwitchDetected, *refusal.ErrorReason)
	})

	t.Run("continues the chain once the published history links the live timeline", func(t *testing.T) {
		t.Parallel()

		decision, err := continueAcrossPromotion(t.Context(), promotionContinuationSpec{
			Extension: extension,
			Decision:  detected,
			PublishHistory: func(context.Context, int) (*catalog.WalHistoryFile, error) {
				return publishedFork, nil
			},
			LoadHistories: func(context.Context) ([]*catalog.WalHistoryFile, error) {
				return []*catalog.WalHistoryFile{publishedFork}, nil
			},
			Logger: slog.New(slog.DiscardHandler),
		})

		require.NoError(t, err)
		require.Equal(t, timelines.TimelineContinueAcrossPromotion, decision.Kind)
	})
}

func TestPublishLineHistories(t *testing.T) {
	t.Parallel()

	storageID := uuid.New()

	t.Run("publishes every timeline from the live one down to the newest backup's", func(t *testing.T) {
		t.Parallel()
		publishedForks := map[int]*catalog.WalHistoryFile{
			2: newPublishedFork(storageID, 2, 1, firstForkLSN),
			3: newPublishedFork(storageID, 3, 2, secondForkLSN),
		}
		publishedTimelines := []int{}

		err := publishLineHistories(
			t.Context(),
			func(_ context.Context, timelineID int) (*catalog.WalHistoryFile, error) {
				publishedTimelines = append(publishedTimelines, timelineID)

				return publishedForks[timelineID], nil
			},
			1,
			3,
		)

		require.NoError(t, err)
		require.Equal(t, []int{3, 2}, publishedTimelines)
	})

	t.Run("stops at a link without a usable fork point", func(t *testing.T) {
		t.Parallel()
		publishedTimelines := []int{}

		err := publishLineHistories(
			t.Context(),
			func(_ context.Context, timelineID int) (*catalog.WalHistoryFile, error) {
				publishedTimelines = append(publishedTimelines, timelineID)

				return withConflictingFork(newPublishedFork(storageID, timelineID, timelineID-1, secondForkLSN)), nil
			},
			1,
			3,
		)

		require.NoError(t, err)
		require.Equal(t, []int{3}, publishedTimelines)
	})

	t.Run("reports a failed publication", func(t *testing.T) {
		t.Parallel()
		publicationErr := errors.New("storage unavailable")

		err := publishLineHistories(t.Context(), func(context.Context, int) (*catalog.WalHistoryFile, error) {
			return nil, publicationErr
		}, 1, 2)

		require.ErrorIs(t, err, publicationErr)
	})
}

func newChainBackup(storageID uuid.UUID, timelineID int, stopLSN walmath.LSN) *catalog.Backup {
	startLSN := stopLSN - 0x100

	return &catalog.Backup{
		ID:         uuid.New(),
		Kind:       catalog.BackupKindFull,
		Status:     catalog.BackupStatusCompleted,
		StorageID:  storageID,
		TimelineID: timelineID,
		StartLSN:   &startLSN,
		StopLSN:    &stopLSN,
	}
}

func newPublishedFork(
	storageID uuid.UUID,
	timelineID int,
	parentTimelineID int,
	forkLSN walmath.LSN,
) *catalog.WalHistoryFile {
	return &catalog.WalHistoryFile{
		ID:               uuid.New(),
		StorageID:        storageID,
		TimelineID:       timelineID,
		HistoryFilename:  walmath.FormatHistoryFilename(uint32(timelineID)),
		ParentTimelineID: &parentTimelineID,
		ForkLSN:          &forkLSN,
	}
}

func withConflictingFork(history *catalog.WalHistoryFile) *catalog.WalHistoryFile {
	history.IsConflicting = true

	return history
}
