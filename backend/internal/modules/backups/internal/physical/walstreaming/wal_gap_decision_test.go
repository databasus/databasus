package walstreaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func TestDecideWalChainGap(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		change   func(*walGapState)
		hasGap   bool
		gapStart walmath.LSN
	}{
		{"reports no gap when this is the first archived segment", func(state *walGapState) { state.PreviousEnd = nil }, false, 0},
		{"reports no gap when the committed predecessor is contiguous", func(state *walGapState) { state.PreviousEnd = new(state.Claim.StartLSN) }, false, 0},
		{"reports no gap when the predecessor is still queued", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 1, endLSN: 100, isComplete: true}}
		}, false, 0},
		{"reports no gap when the predecessor has an uncommitted catalog claim", func(state *walGapState) { state.HasPredecessor = true }, false, 0},
		{"reports a gap when no predecessor can still arrive", func(*walGapState) {}, true, 10},
		{"reports a gap when the queued predecessor is incomplete", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 1, endLSN: 100, isComplete: false}}
		}, true, 10},
		{"reports a gap when a queued predecessor belongs to another timeline", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 2, endLSN: 100, isComplete: true}}
		}, true, 10},
		{"starts the gap after the highest complete queued segment when it precedes the hole", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 1, endLSN: 50, isComplete: true}}
		}, true, 50},
		{"ignores a short queued segment when choosing the gap start", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 1, endLSN: 50, isComplete: false}}
		}, true, 10},
		{"keeps the committed position when queued segments end before it", func(state *walGapState) {
			state.QueuedSegments = []queuedWalSegment{{timelineID: 1, endLSN: 5, isComplete: true}}
		}, true, 10},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := walGapState{
				PreviousEnd: new(walmath.LSN(10)),
				Claim:       &catalog.WalSegment{TimelineID: 1, StartLSN: 100},
			}
			testCase.change(&state)

			gap := decideWalChainGap(state)

			if !testCase.hasGap {
				assert.Nil(t, gap)
				return
			}
			require.NotNil(t, gap)
			assert.Equal(t, testCase.gapStart, gap.Start)
			assert.Equal(t, walmath.LSN(100), gap.End)
		})
	}
}

func TestWalGapSequence(t *testing.T) {
	t.Run("reports one gap when several contiguous segments follow a missing segment", func(t *testing.T) {
		var previousEnd *walmath.LSN
		gaps := []walChainGap{}
		for _, segmentNo := range []uint64{20, 22, 23, 24} {
			start := walmath.LSN(segmentNo * uint64(testWalSegmentSize))
			gap := decideWalChainGap(
				walGapState{
					PreviousEnd: previousEnd,
					Claim:       &catalog.WalSegment{TimelineID: 1, StartLSN: start},
				},
			)
			if gap != nil {
				gaps = append(gaps, *gap)
			}
			previousEnd = new(start + walmath.LSN(testWalSegmentSize))
		}
		assert.Equal(t, []walChainGap{{Start: segmentStartLSN(21), End: segmentStartLSN(22)}}, gaps)
	})
	t.Run("reports both holes when a lower gap is found after a higher gap", func(t *testing.T) {
		higher := decideWalChainGap(
			walGapState{
				PreviousEnd:    new(segmentStartLSN(29)),
				Claim:          &catalog.WalSegment{TimelineID: 1, StartLSN: segmentStartLSN(40)},
				QueuedSegments: []queuedWalSegment{{timelineID: 1, endLSN: segmentStartLSN(31), isComplete: true}},
			},
		)
		lower := decideWalChainGap(
			walGapState{
				PreviousEnd: new(segmentStartLSN(29)),
				Claim:       &catalog.WalSegment{TimelineID: 1, StartLSN: segmentStartLSN(30)},
			},
		)

		require.NotNil(t, higher)
		require.NotNil(t, lower)
		assert.Equal(t, walChainGap{Start: segmentStartLSN(31), End: segmentStartLSN(40)}, *higher)
		assert.Equal(t, walChainGap{Start: segmentStartLSN(29), End: segmentStartLSN(30)}, *lower)
	})
}
