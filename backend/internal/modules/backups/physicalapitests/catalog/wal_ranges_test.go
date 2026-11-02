package catalog_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPhysicalBackupWalRanges(t *testing.T) {
	t.Parallel()

	type walRangeCase struct {
		stopSegment uint64
		count       int
		hasGap      bool
	}

	testCases := []struct {
		name           string
		segmentNumbers []uint64
		beforeMinute   int
		ranges         map[uint64]walRangeCase
	}{
		{
			name:           "preserves a gap when a lower segment arrives after the gap",
			segmentNumbers: []uint64{2, 5, 3},
			ranges: map[uint64]walRangeCase{
				2: {4, 2, false}, 5: {6, 1, true},
			},
		},
		{
			name:           "merges each side of a gap when segments arrive out of order",
			segmentNumbers: []uint64{2, 6, 5, 3},
			ranges: map[uint64]walRangeCase{
				2: {4, 2, false}, 5: {7, 2, true},
			},
		},
		{
			name:           "merges contiguous segments when they arrive out of order",
			segmentNumbers: []uint64{4, 2, 3},
			ranges: map[uint64]walRangeCase{
				2: {5, 3, false},
			},
		},
		{
			name:           "splits filtered ranges without a catalog gap when a middle segment is excluded",
			segmentNumbers: []uint64{2, 4, 3},
			beforeMinute:   2,
			ranges: map[uint64]walRangeCase{
				2: {3, 1, false}, 4: {5, 1, false},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			prereqs := newCatalog(t)
			receivedAt := time.Now().UTC().Add(-time.Hour)
			for index, segmentNumber := range testCase.segmentNumbers {
				seedWalSegmentReceivedAt(
					t,
					prereqs,
					segmentNumber,
					receivedAt.Add(time.Duration(index)*time.Minute),
				)
			}

			query := "?type=WAL"
			if testCase.beforeMinute != 0 {
				cutoff := receivedAt.Add(time.Duration(testCase.beforeMinute)*time.Minute - time.Second)
				query += "&createdBefore=" + cutoff.Format(time.RFC3339Nano)
			}

			page := getBackups(t, prereqs, query)
			require.Len(t, page.Backups, len(testCase.ranges))
			for _, walRange := range page.Backups {
				var startSegment uint64
				for segmentNumber := range testCase.ranges {
					if walRange.StartLSN == fixtureLSN(segmentNumber*segmentBytes).String() {
						startSegment = segmentNumber
						break
					}
				}
				segmentRange, hasRange := testCase.ranges[startSegment]
				require.True(t, hasRange, "unexpected range start: %s", walRange.StartLSN)
				assert.Equal(t, fixtureLSN(segmentRange.stopSegment*segmentBytes).String(), walRange.StopLSN)
				assert.Equal(t, segmentRange.count, *walRange.WalSegmentsCount)
				assert.Equal(t, segmentRange.hasGap, walRange.HasGapBefore)
			}
		})
	}
}
