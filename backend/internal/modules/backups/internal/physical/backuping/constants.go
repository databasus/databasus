package backuping

import "time"

const (
	bastionOpenTimeout            = 30 * time.Second
	clusterSizeProbeTimeout       = 30 * time.Second
	maxConsecutiveFailedIncrTries = 3
	recentFullAttemptsWindow      = 3
)
