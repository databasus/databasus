package verifications

import (
	"time"

	"databasus-backend/internal/modules/verification/internal/agents"
)

func hasFreshAcceptedHeartbeat(agent *agents.Agent, now time.Time) bool {
	return agent.LastSeenAt != nil && !agent.LastSeenAt.Before(now.Add(-staleAgentThreshold))
}
