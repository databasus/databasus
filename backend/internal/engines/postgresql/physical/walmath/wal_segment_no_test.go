package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestWalSegmentNo(t *testing.T) {
	t.Parallel()
	t.Run("uses the cluster size with default and custom segments", func(t *testing.T) {
		t.Parallel()
		for _, segmentSize := range []uint64{walmath.DefaultSegmentSizeBytes, 64 * 1024 * 1024} {
			require.Equal(t, walmath.WalSegmentNo(0), walmath.NewWalSegmentNo(walmath.LSN(segmentSize-1), segmentSize))
			require.Equal(t, walmath.WalSegmentNo(1), walmath.NewWalSegmentNo(walmath.LSN(segmentSize), segmentSize))
			require.Equal(
				t,
				walmath.WalSegmentNo(3),
				walmath.NewWalSegmentNo(walmath.LSN(3*segmentSize+12345), segmentSize),
			)
			segment := walmath.WalSegmentNo(0x101)
			require.Equal(t, walmath.LSN(0x101*segmentSize), segment.FirstLSN(segmentSize))
			require.Equal(t, walmath.WalSegmentNo(0x102), segment.Next())
			timeline, parsedSegment, err := walmath.ParseWALFilename(segment.GetFilename(7, segmentSize), segmentSize)
			require.NoError(t, err)
			require.Equal(t, uint32(7), timeline)
			require.Equal(t, uint64(segment), parsedSegment)
		}
	})
}
