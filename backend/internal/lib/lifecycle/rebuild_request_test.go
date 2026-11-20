package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebuildRequest(t *testing.T) {
	t.Parallel()

	t.Run("keeps one request until the process takes it", func(t *testing.T) {
		t.Parallel()
		rebuildRequest := NewRebuildRequest()

		rebuildRequest.Request()
		rebuildRequest.Request()

		assert.Len(t, rebuildRequest.GetSignal(), 1)
		<-rebuildRequest.GetSignal()
		assert.Empty(t, rebuildRequest.GetSignal())
	})
}
