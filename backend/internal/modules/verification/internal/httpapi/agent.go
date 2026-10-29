package httpapi

import "databasus-backend/internal/modules/verification/internal/agents"

func fromAgent(agent agents.Agent) VerificationAgent {
	return VerificationAgent{
		Id:                agent.ID,
		Name:              agent.Name,
		CreatedAt:         agent.CreatedAt,
		LastSeenAt:        agent.LastSeenAt,
		MaxCpu:            agent.MaxCPU,
		MaxRamGb:          agent.MaxRAMGb,
		MaxDiskGb:         agent.MaxDiskGb,
		MaxConcurrentJobs: agent.MaxConcurrentJobs,
	}
}

func fromAgents(agents []agents.Agent) []VerificationAgent {
	items := make([]VerificationAgent, len(agents))
	for index, agent := range agents {
		items[index] = fromAgent(agent)
	}
	return items
}
