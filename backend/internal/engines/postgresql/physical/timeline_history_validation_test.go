package physical

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestValidateTimelineHistory(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name     string
		body     string
		filename string
		isValid  bool
	}{
		{
			name:    "accepts history with ancestor entries",
			body:    "1\t0/100\tfirst promotion\n2\t0/200\tsecond promotion\n",
			isValid: true,
		},
		{
			name:    "accepts history with comments and equal fork positions",
			body:    "# note\n\n1\t0/100\tfirst\n2\t0/100\tsecond\n",
			isValid: true,
		},
		{name: "accepts history without a reason", body: "1\t0/100\n", isValid: true},
		{name: "rejects history when content is empty"},
		{name: "rejects history with only comments", body: "# note\n"},
		{name: "rejects history with zero ancestor", body: "0\t0/100\treason\n"},
		{name: "rejects history with current ancestor", body: "3\t0/100\treason\n"},
		{name: "rejects history with repeated ancestor", body: "1\t0/100\n1\t0/200\n"},
		{name: "rejects history with decreasing fork positions", body: "1\t0/200\n2\t0/100\n"},
		{name: "rejects history with malformed position", body: "1\t0/100junk\n"},
		{name: "rejects history with overflowing position", body: "1\t100000000/0\n"},
		{name: "rejects history with missing position", body: "1\n"},
		{name: "rejects history with another filename", body: "1\t0/100\n", filename: "00000002.history"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			filename := scenario.filename
			if filename == "" {
				filename = "00000003.history"
			}

			_, err := validateTimelineHistory(
				3,
				TimelineHistory{Filename: filename, Body: []byte(scenario.body), Fork: TimelineFork{}},
			)

			if scenario.isValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInvalidTimelineHistory)
			}
		})
	}
}

func TestValidateTimelineHistoryFork(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name         string
		body         string
		expectedFork TimelineFork
	}{
		{
			name:         "returns the only entry as the fork",
			body:         "1\t0/3000148\tno recovery target specified\n",
			expectedFork: TimelineFork{ParentTimelineID: 1, ForkLSN: walmath.LSN(0x3000148)},
		},
		{
			name:         "returns the last entry when the history has two promotions",
			body:         "1\t0/3000148\tfirst\n2\t1/A0000000\tsecond\n",
			expectedFork: TimelineFork{ParentTimelineID: 2, ForkLSN: walmath.LSN(0x1A0000000)},
		},
		{
			name:         "returns a fork on a segment boundary",
			body:         "# note\n1\t0/5000000\n",
			expectedFork: TimelineFork{ParentTimelineID: 1, ForkLSN: walmath.LSN(0x5000000)},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			fork, err := validateTimelineHistory(
				3,
				TimelineHistory{Filename: "00000003.history", Body: []byte(scenario.body), Fork: TimelineFork{}},
			)

			require.NoError(t, err)
			require.Equal(t, scenario.expectedFork, fork)
		})
	}
}
