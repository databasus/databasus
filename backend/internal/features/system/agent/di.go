package system_agent

import "sync"

var getAgentController = sync.OnceValue(func() *AgentController {
	return &AgentController{}
})

func GetAgentController() *AgentController {
	return getAgentController()
}
