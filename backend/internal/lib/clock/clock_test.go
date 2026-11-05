package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystem(t *testing.T) {
	t.Parallel()

	t.Run("returns the time in utc", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, time.UTC, NewSystem().Now().Location())
	})
}
