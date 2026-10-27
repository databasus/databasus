package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestParseWALFilename(t *testing.T) {
	t.Parallel()
	t.Run("extracts timeline and segment number with valid names", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name               string
			expectedTimelineID uint32
			expectedLogSegNo   uint64
		}{
			{"000000010000000100000001", 1, 1<<8 + 1},
			{"000000100000020000000030", 1 << 4, 2<<16 + 3<<4},
			{"10000000f0000000000000a0", 1 << 28, 15<<36 + 10<<4},
			{"ffffffffffffffff000000ff", 1<<32 - 1, 1<<40 - 1},
		}

		for _, filenameCase := range cases {
			timeline, segNo, err := walmath.ParseWALFilename(filenameCase.name, walmath.DefaultSegmentSizeBytes)
			require.NoError(t, err, filenameCase.name)
			assert.Equal(t, filenameCase.expectedTimelineID, timeline, filenameCase.name)
			assert.Equal(t, filenameCase.expectedLogSegNo, segNo, filenameCase.name)
		}
	})
}

func TestParseWALFilenameMalformed(t *testing.T) {
	t.Parallel()
	t.Run("returns an error with malformed names", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			"00000001",
			"000000010000000100000100000000010000000100000001",
			"000000010000000100000100",
			"000000010000000110000001",
			"000xYz010000000100000001",
			"0000000100xYz00100000001",
			"000000010000000100xYz001",
		}

		for _, filenameCase := range cases {
			_, _, err := walmath.ParseWALFilename(filenameCase, walmath.DefaultSegmentSizeBytes)
			require.Error(t, err, filenameCase)
		}
	})
}

func TestParseWALFilenameCustomSize(t *testing.T) {
	t.Parallel()
	t.Run("uses the cluster boundary with custom segment sizes", func(t *testing.T) {
		t.Parallel()

		const segmentSize = uint64(64 * 1024 * 1024)

		_, _, err := walmath.ParseWALFilename("10000000f0000000000000a0", segmentSize)
		require.Error(t, err)

		cases := []struct {
			name               string
			expectedTimelineID uint32
			expectedLogSegNo   uint64
		}{
			{"000000010000000000000001", 1, 1},
			{"000000100000000100000001", 1 << 4, 4<<4 + 1},
			{"000000100000020000000030", 1 << 4, 2<<14 + 3<<4},
			{"10000000f000000000000030", 1 << 28, 15<<34 + 3<<4},
		}

		for _, filenameCase := range cases {
			timeline, segNo, err := walmath.ParseWALFilename(filenameCase.name, segmentSize)
			require.NoError(t, err, filenameCase.name)
			assert.Equal(t, filenameCase.expectedTimelineID, timeline, filenameCase.name)
			assert.Equal(t, filenameCase.expectedLogSegNo, segNo, filenameCase.name)
		}
	})
}

func TestParseWALFilenameIndependentSizes(t *testing.T) {
	t.Parallel()
	t.Run("uses the supplied size with a non default cluster", func(t *testing.T) {
		t.Parallel()

		timelineID, segmentNo, err := walmath.ParseWALFilename(
			"000000010000000200000003",
			64*1024*1024,
		)

		require.NoError(t, err)
		assert.Equal(t, uint32(1), timelineID)
		assert.Equal(t, uint64(2*64+3), segmentNo)
	})
}

func TestIsWalFilename(t *testing.T) {
	t.Parallel()
	t.Run("returns true with valid names", func(t *testing.T) {
		t.Parallel()

		assert.True(t, walmath.IsWalFilename("000000010000000100000001", walmath.DefaultSegmentSizeBytes))
		assert.True(t, walmath.IsWalFilename("ffffffffffffffff000000ff", walmath.DefaultSegmentSizeBytes))
	})
}

func TestIsWalFilenameMalformed(t *testing.T) {
	t.Parallel()
	t.Run("returns false with invalid names", func(t *testing.T) {
		t.Parallel()

		assert.False(t, walmath.IsWalFilename("", walmath.DefaultSegmentSizeBytes))
		assert.False(t, walmath.IsWalFilename("not-a-wal-name", walmath.DefaultSegmentSizeBytes))
		assert.False(t, walmath.IsWalFilename("000000010000000100000001.partial", walmath.DefaultSegmentSizeBytes))
	})
}

func TestFormatHistoryFilename(t *testing.T) {
	t.Parallel()
	t.Run("formats a known timeline as a postgres history filename", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "00000003.history", walmath.FormatHistoryFilename(3))
		assert.Equal(t, "0000000A.history", walmath.FormatHistoryFilename(10))
	})
}

func TestParseWALFilenameTimeline(t *testing.T) {
	t.Parallel()
	t.Run("reads the timeline without the segment size", func(t *testing.T) {
		t.Parallel()

		// A cluster with segments smaller than 16 MB numbers them past what 16 MB
		// arithmetic accepts. The timeline field reads either way.
		segmentOfASmallerSegmentedCluster := "000000020000000000000100"

		timeline, err := walmath.ParseWALFilenameTimeline(segmentOfASmallerSegmentedCluster)
		require.NoError(t, err)
		assert.Equal(t, uint32(2), timeline)

		_, _, err = walmath.ParseWALFilename(segmentOfASmallerSegmentedCluster, 16*1024*1024)
		require.Error(t, err, "the segment-size-aware parser is the one that rejects such a name")
	})
}

func TestParseWALFilenameTimelineMalformed(t *testing.T) {
	t.Parallel()
	t.Run("returns an error with malformed names", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			"",
			"00000002",
			"000xYz020000000000000030",
			"00000002000xYz0000000030",
			"0000000200000000000xYz30",
			"0000000200000000000000300",
		}

		for _, name := range cases {
			_, err := walmath.ParseWALFilenameTimeline(name)
			require.Error(t, err, name)
		}
	})
}

func TestHistoryFilenameRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("round trips the timeline with a history filename", func(t *testing.T) {
		t.Parallel()
		for _, timeline := range []uint32{1, 10, 0xffffffff} {
			parsedTimeline, err := walmath.ParseHistoryFilename(walmath.FormatHistoryFilename(timeline))
			require.NoError(t, err)
			require.Equal(t, int(timeline), parsedTimeline)
		}
		for _, filename := range []string{"", "00000001", "wrong.history"} {
			_, err := walmath.ParseHistoryFilename(filename)
			require.Error(t, err)
		}
	})
}
