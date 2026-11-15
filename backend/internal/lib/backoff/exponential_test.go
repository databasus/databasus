package backoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExponential(t *testing.T) {
	t.Parallel()

	policy := Exponential{First: time.Minute, Max: 15 * time.Minute}

	for _, testCase := range []struct {
		name          string
		attemptCount  int
		expectedDelay time.Duration
	}{
		{name: "waits the first delay after the first attempt", attemptCount: 1, expectedDelay: time.Minute},
		{name: "doubles the delay after each attempt", attemptCount: 3, expectedDelay: 4 * time.Minute},
		{name: "stops at the maximum", attemptCount: 5, expectedDelay: 15 * time.Minute},
		{name: "stops at the maximum without overflowing", attemptCount: 500, expectedDelay: 15 * time.Minute},
		{name: "counts an attempt below one as the first", attemptCount: 0, expectedDelay: time.Minute},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expectedDelay, policy.GetDelay(testCase.attemptCount))
		})
	}
}
