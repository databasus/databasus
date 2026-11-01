package walstreaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestSegmentBounds(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		filename     string
		segmentSize  int64
		start        walmath.LSN
		nextFilename string
	}{
		{"keeps positions contiguous across a log ID boundary with default segments", "0000000100000000000000FF", 16 * 1024 * 1024, 0xFF000000, "000000010000000100000000"},
		{"uses the source segment size when segments contain 64 MiB", "000000010000000200000003", 64 * 1024 * 1024, (2 << 32) + 3*64*1024*1024, "000000010000000200000004"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			timeline, start, end, err := segmentBounds(testCase.filename, testCase.segmentSize)
			require.NoError(t, err)

			assert.Equal(t, 1, timeline)
			assert.Equal(t, testCase.start, start)
			assert.Equal(t, start+walmath.LSN(testCase.segmentSize), end)

			_, nextStart, _, err := segmentBounds(testCase.nextFilename, testCase.segmentSize)
			require.NoError(t, err)
			assert.Equal(t, end, nextStart)
		})
	}
}
