package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDiskExhausted(t *testing.T) {
	t.Run("detects enospc signatures", func(t *testing.T) {
		assert.True(t, IsDiskExhausted("pg_restore: error: could not write to output file: No space left on device"))
		assert.True(t, IsDiskExhausted("ERROR: could not extend file"))
		assert.False(t, IsDiskExhausted("pg_restore: error: relation already exists"))
		assert.False(t, IsDiskExhausted(""))
	})
}
