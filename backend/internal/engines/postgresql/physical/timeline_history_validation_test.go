package physical

import (
	"testing"

	"github.com/stretchr/testify/require"
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

			err := validateTimelineHistory(3, TimelineHistory{Filename: filename, Body: []byte(scenario.body)})

			if scenario.isValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInvalidTimelineHistory)
			}
		})
	}
}
