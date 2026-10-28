package walstreaming

import "sync/atomic"

type receiverRespawnCounter struct {
	restarts atomic.Int64
	pauses   atomic.Int64
}

func (c *receiverRespawnCounter) recordRespawn(isAfterBackpressurePause bool) {
	if isAfterBackpressurePause {
		c.pauses.Add(1)

		return
	}

	c.restarts.Add(1)
}

func (c *receiverRespawnCounter) takeCounts() (restarts, pauses int) {
	return int(c.restarts.Swap(0)), int(c.pauses.Swap(0))
}
