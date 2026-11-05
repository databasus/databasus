package walstreaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWalLiveStateCounters(t *testing.T) {
	t.Run("keeps restart counts for the next sample when a live snapshot is read", func(t *testing.T) {
		supervisor := &Streamer{watchDir: t.TempDir()}
		supervisor.receiverRespawns.recordRespawn(false)
		supervisor.getLiveStateForArchive(walLiveArchive{MeasuredAt: time.Now().UTC()})
		restarts, _ := supervisor.receiverRespawns.takeCounts()
		assert.Equal(t, 1, restarts)
	})
}
