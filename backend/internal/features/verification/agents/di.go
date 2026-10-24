package verification_agents

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var getAgentRepository = sync.OnceValue(func() *AgentRepository {
	return &AgentRepository{}
})

var getAgentService = sync.OnceValue(func() *AgentService {
	return &AgentService{
		getAgentRepository(),
		audit_logs.GetAuditLogService(),
		ratelimiter.GetCounter(),
		logger.GetLogger(),
		nil,
	}
})

var getAgentController = sync.OnceValue(func() *AgentController {
	return &AgentController{
		getAgentService(),
	}
})

var getAgentFacingController = sync.OnceValue(func() *AgentFacingController {
	return &AgentFacingController{
		getAgentService(),
	}
})

func GetAgentService() *AgentService {
	return getAgentService()
}

func GetAgentController() *AgentController {
	return getAgentController()
}

func GetAgentFacingController() *AgentFacingController {
	return getAgentFacingController()
}
