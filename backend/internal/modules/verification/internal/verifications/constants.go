package verifications

import "time"

const (
	MaxAgentSideAttempts = 3
	minRestoredSizeRatio = 0.20
	staleAgentThreshold  = 5 * time.Minute
	agentJobReclaimGrace = 90 * time.Second
	maxPendingDuration   = 24 * time.Hour
)

const (
	cancelMessageByUser             = "verification canceled by user"
	cancelMessageScheduleDisabled   = "verification canceled: its verification schedule was disabled"
	cancelMessageSupersededByBackup = "verification canceled: superseded by a newer backup's scheduled verification"
)
