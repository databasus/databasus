package backuping

import "time"

const (
	bastionOpenTimeout            = 30 * time.Second
	clusterSizeProbeTimeout       = 30 * time.Second
	maxConsecutiveFailedIncrTries = 3
	recentFullAttemptsWindow      = 3
	requestSchedulerInterval      = time.Second
	schedulerInterval             = 15 * time.Second
)
